package embed_util

import (
	"embed"
	"io/fs"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

// ExtractEmbeddedFiles 解压嵌入的文件到指定目录
func ExtractEmbeddedFiles(embeddedFS embed.FS, sourceDir string, targetDir string) error {
	log.Println("ExtractEmbeddedFiles,sourceDir:", sourceDir, "targetDir:", targetDir)
	// 创建目标目录
	err := os.MkdirAll(targetDir, os.ModePerm)
	if err != nil {
		return err
	}
	// 遍历嵌入的文件
	err = fs.WalkDir(embeddedFS, sourceDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if !d.IsDir() {
			// 读取嵌入的文件内容
			data, err := embeddedFS.ReadFile(path)
			if err != nil {
				return err
			}

			// 确定目标文件路径
			targetPath := filepath.Join(targetDir, filepath.Base(path))

			// 写入目标文件
			err = ioutil.WriteFile(targetPath, data, os.ModePerm)
			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}
