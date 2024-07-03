package assets

import (
	"embed"
	log "github.com/sirupsen/logrus"
	"mydocker/embed_util"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
)

//go:embed busybox.tar
var embedFS embed.FS

// ExtractBusybox 返回值为工作目录
func ExtractBusybox(targetDir string) (error, string) {
	sourceFile := "busybox.tar"
	log.Info("Extracting busybox to ", targetDir)
	err := embed_util.ExtractEmbeddedFiles(embedFS, sourceFile, targetDir)
	if err != nil {
		return err, ""
	}
	return nil, filepath.Join(targetDir, sourceFile)
}

func GetHomeDir() string {
	usr, err := user.Current()
	if err != nil {
		log.Fatal(err)
	}
	return usr.HomeDir
}

// UntarFile 解压tar文件到指定目录
func UntarFile(tarFile, targetDir string) error {
	// 构建 tar 命令
	cmd := exec.Command("tar", "-xf", tarFile, "-C", targetDir)

	// 捕获命令的标准输出和错误输出
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// 执行命令
	err := cmd.Run()
	if err != nil {
		return err
	}

	log.Printf("已成功解压 %s 到 %s", tarFile, targetDir)
	return nil
}
