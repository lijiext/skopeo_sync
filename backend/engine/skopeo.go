package engine

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"os/exec"
	"strings"
)

// SyncImage 执行 skopeo copy 同步镜像
// 增加了对 srcUser, srcPass, destUser, destPass 的认证支持，并支持动态日志回调
func SyncImage(srcUrl, destUrl, srcImage, destImage string, retries int, srcUser, srcPass, destUser, destPass string, logCallback func(string)) error {
	src := fmt.Sprintf("docker://%s/%s", srcUrl, srcImage)
	dest := fmt.Sprintf("docker://%s/%s", destUrl, destImage)

	args := []string{"copy", "--all", "--retry-times", fmt.Sprintf("%d", retries)}

	if srcUser != "" && srcPass != "" {
		args = append(args, "--src-creds", fmt.Sprintf("%s:%s", srcUser, srcPass))
	}
	if destUser != "" && destPass != "" {
		args = append(args, "--dest-creds", fmt.Sprintf("%s:%s", destUser, destPass))
	}

	args = append(args, src, dest)
	cmd := exec.Command("skopeo", args...)

	// 获取 stdout 和 stderr 的管道
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("skopeo 启动失败: %v", err)
	}

	// 开启 goroutine 实时读取并回调日志
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			logCallback(scanner.Text() + "\n")
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			logCallback(scanner.Text() + "\n")
		}
	}()

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("skopeo 执行失败: %v", err)
	}
	return nil
}

// GetImageSize 获取目标库中该镜像占用的实际总流量大小（单位：字节）
// 使用 skopeo inspect 的标准输出解析 size
func GetImageSize(destUrl, destImage, destUser, destPass string) int64 {
	dest := fmt.Sprintf("docker://%s/%s", destUrl, destImage)
	// 不要用 --raw，用普通的 inspect 来获取全量的聚合信息或者用正则提取
	// skopeo inspect docker://...
	args := []string{"inspect"}
	if destUser != "" && destPass != "" {
		args = append(args, "--creds", fmt.Sprintf("%s:%s", destUser, destPass))
	}
	args = append(args, dest)

	cmd := exec.Command("skopeo", args...)
	out, err := cmd.Output()
	if err != nil {
		return 0
	}

	// 解析 inspect 的 JSON 获取全量架构大小或者简单从层大小相加
	// 普通的 skopeo inspect 会返回所有架构架构，这里简化依然通过正则提取整个 JSON 中的所有的 Size/size 字段值并加起来，
	// 这可能会偏大（因为有些层共用），但能反映整体推送到目标库消耗的逻辑容量
	var totalSize int64
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		lowerLine := strings.ToLower(line)
		if strings.Contains(lowerLine, "\"size\":") {
			var size int64
			cleanLine := strings.TrimSpace(lowerLine)
			cleanLine = strings.TrimSuffix(cleanLine, ",")
			fmt.Sscanf(cleanLine, "\"size\": %d", &size)
			totalSize += size
		}
	}

	return totalSize
}

// VerifyImage 验证源和目标镜像的 Digest 是否一致
// 使用 --raw 获取最外层 Manifest List (Index) 的原始内容进行 SHA256 比对
// 解决 macOS 等宿主机上找不到特定架构/OS 子镜像导致 inspect 报错的问题
func VerifyImage(srcUrl, destUrl, srcImage, destImage string, srcUser, srcPass, destUser, destPass string) (bool, string) {
	src := fmt.Sprintf("docker://%s/%s", srcUrl, srcImage)
	dest := fmt.Sprintf("docker://%s/%s", destUrl, destImage)

	var logOutput string

	// ========== 1. 获取源镜像 manifest list 原始数据 ==========
	srcArgs := []string{"inspect", "--raw"}
	if srcUser != "" && srcPass != "" {
		srcArgs = append(srcArgs, "--creds", fmt.Sprintf("%s:%s", srcUser, srcPass))
	}
	srcArgs = append(srcArgs, src)
	
	logOutput += fmt.Sprintf("执行命令: skopeo %s\n", strings.Join(srcArgs, " "))
	
	srcCmd := exec.Command("skopeo", srcArgs...)
	srcOut, err := srcCmd.CombinedOutput()
	if err != nil {
		logOutput += fmt.Sprintf("获取源 Raw Manifest 失败: %v\n", err)
		logOutput += fmt.Sprintf("Skopeo 输出详情:\n%s\n", string(srcOut))
		return false, logOutput
	}
	
	// 计算源镜像 Manifest 的 SHA256 散列值
	srcDigestStr := fmt.Sprintf("sha256:%x", sha256.Sum256(srcOut))
	logOutput += fmt.Sprintf("源 Manifest SHA256: %s\n", srcDigestStr)

	// ========== 2. 获取目标镜像 manifest list 原始数据 ==========
	destArgs := []string{"inspect", "--raw"}
	if destUser != "" && destPass != "" {
		destArgs = append(destArgs, "--creds", fmt.Sprintf("%s:%s", destUser, destPass))
	}
	destArgs = append(destArgs, dest)
	
	logOutput += fmt.Sprintf("执行命令: skopeo %s\n", strings.Join(destArgs, " "))
	
	destCmd := exec.Command("skopeo", destArgs...)
	destOut, err := destCmd.CombinedOutput()
	if err != nil {
		logOutput += fmt.Sprintf("获取目标 Raw Manifest 失败: %v\n", err)
		logOutput += fmt.Sprintf("Skopeo 输出详情:\n%s\n", string(destOut))
		return false, logOutput
	}
	
	// 计算目标镜像 Manifest 的 SHA256 散列值
	destDigestStr := fmt.Sprintf("sha256:%x", sha256.Sum256(destOut))
	logOutput += fmt.Sprintf("目标 Manifest SHA256: %s\n", destDigestStr)

	// ========== 3. 校验比对 ==========
	match := (srcDigestStr == destDigestStr)
	if match {
		logOutput += "校验结果: 匹配\n"
	} else {
		logOutput += "校验结果: 不匹配\n"
	}
	
	return match, logOutput
}
