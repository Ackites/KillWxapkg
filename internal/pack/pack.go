package pack

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

func Repack(path string, watch bool, outputDir string) {
	// 过滤空白字符
	path = strings.TrimSpace(path)
	outputDir = strings.TrimSpace(outputDir)

	// 如果是目录，则打包目录
	if fileInfo, err := os.Stat(path); err != nil || !fileInfo.IsDir() {
		log.Printf("错误: %s 不是一个有效的目录\n", path)
		return
	}

	// 打包目录
	err := packWxapkg(path, outputDir)
	if err != nil {
		log.Printf("错误: %v\n", err)
		return
	}

	if watch {
		watchDir(path, outputDir)
	}

	return
}

type WxapkgFile struct {
	NameLen uint32
	Name    string
	Offset  uint32
	Size    uint32
}

// 打包文件到 wxapkg 格式
func packWxapkg(inputDir string, outputDir string) error {
	var files []WxapkgFile
	var totalSize uint32

	// 检查 outputDir 是否存在及其类型
	outputInfo, err := os.Stat(outputDir)
	outputFile := ""
	if err != nil {
		if os.IsNotExist(err) {
			// 如果 outputDir 不存在，假设它是一个文件路径或一个需要创建的目录
			if filepath.Ext(outputDir) == "" {
				// 创建目录
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return fmt.Errorf("无法创建输出目录: %w", err)
				}
				outputFile = filepath.Join(outputDir, "output.wxapkg")
			} else {
				outputFile = outputDir
			}
		} else {
			return fmt.Errorf("无法访问输出目录: %w", err)
		}
	} else {
		if outputInfo.IsDir() {
			outputFile = filepath.Join(outputDir, "output.wxapkg")
		} else {
			outputFile = outputDir
		}
	}

	// 计算文件列表
	err = filepath.Walk(inputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 排除目录
		if info.IsDir() {
			return nil
		}

		// 排除 .wxapkg 文件
		if filepath.Ext(path) == ".wxapkg" {
			return nil
		}

		relPath, _ := filepath.Rel(inputDir, path)
		relPath = "/" + filepath.ToSlash(relPath) // 确保路径以 '/' 开头，并且使用 Unix 风格的路径分隔符
		size := uint32(info.Size())

		files = append(files, WxapkgFile{
			NameLen: uint32(len(relPath)),
			Name:    relPath,
			Offset:  totalSize, // 预计算文件偏移量
			Size:    size,
		})

		totalSize += size
		return nil
	})
	if err != nil {
		return fmt.Errorf("计算文件列表失败: %w", err)
	}

	// 创建输出文件
	outFile, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("创建输出文件失败: %w", err)
	}
	defer func(outFile *os.File) {
		err := outFile.Close()
		if err != nil {
			log.Printf("关闭输出文件失败: %v\n", err)
		}
	}(outFile)

	// 写入文件头
	if err := binary.Write(outFile, binary.BigEndian, byte(0xBE)); err != nil {
		return fmt.Errorf("写入文件头标记失败: %w", err)
	}

	info1 := uint32(0) // 示例值
	if err := binary.Write(outFile, binary.BigEndian, info1); err != nil {
		return fmt.Errorf("写入 info1 失败: %w", err)
	}

	// 计算索引段长度，包含每个文件的元数据长度和文件名长度
	var indexInfoLength uint32
	for _, file := range files {
		indexInfoLength += 4 + uint32(len(file.Name)) + 4 + 4 // NameLen + Name + Offset + Size
	}

	if err := binary.Write(outFile, binary.BigEndian, indexInfoLength); err != nil {
		return fmt.Errorf("写入索引段长度失败: %w", err)
	}

	bodyInfoLength := totalSize
	if err := binary.Write(outFile, binary.BigEndian, bodyInfoLength); err != nil {
		return fmt.Errorf("写入数据段长度失败: %w", err)
	}

	if err := binary.Write(outFile, binary.BigEndian, byte(0xED)); err != nil {
		return fmt.Errorf("写入文件尾标记失败: %w", err)
	}

	// 写入文件数量
	fileCount := uint32(len(files))
	if err := binary.Write(outFile, binary.BigEndian, fileCount); err != nil {
		return fmt.Errorf("写入文件数量失败: %w", err)
	}

	// 写入索引段
	for _, file := range files {
		if err := binary.Write(outFile, binary.BigEndian, file.NameLen); err != nil {
			return fmt.Errorf("写入文件名长度失败: %w", err)
		}
		if _, err := outFile.Write([]byte(file.Name)); err != nil {
			return fmt.Errorf("写入文件名失败: %w", err)
		}
		// 加上 18 字节文件头长度和索引段长度
		if err := binary.Write(outFile, binary.BigEndian, file.Offset+indexInfoLength+18); err != nil {
			return fmt.Errorf("写入文件偏移量失败: %w", err)
		}
		if err := binary.Write(outFile, binary.BigEndian, file.Size); err != nil {
			return fmt.Errorf("写入文件大小失败: %w", err)
		}
	}

	// 写入数据段
	for _, file := range files {
		func() {
			f, err := os.Open(filepath.Join(inputDir, file.Name[1:])) // 去掉路径开头的 '/' 以正确打开文件
			if err != nil {
				log.Printf("打开文件失败: %v\n", err)
				return
			}
			defer func(f *os.File) {
				err := f.Close()
				if err != nil {
					log.Printf("关闭文件失败: %v\n", err)
				}
			}(f)

			if _, err = io.Copy(outFile, f); err != nil {
				log.Printf("写入文件内容失败: %v\n", err)
			}
		}()
	}

	return nil
}

func watchDir(inputDir string, outputDir string) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Println("ERROR: ", err)
		return
	}
	defer func(watcher *fsnotify.Watcher) {
		err := watcher.Close()
		if err != nil {
			log.Println("ERROR: ", err)
		}
	}(watcher)

	done := make(chan bool)
	go func() {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create || event.Op&fsnotify.Remove == fsnotify.Remove {
					log.Println("检测到文件变化: ", event.Name)
					if err := packWxapkg(inputDir, outputDir); err != nil {
						log.Println("打包失败: ", err)
					} else {
						log.Println("打包成功")
					}
				}
			case err := <-watcher.Errors:
				log.Println("ERROR: ", err)
			}
		}
	}()

	err = watcher.Add(inputDir)
	if err != nil {
		log.Println("ERROR: ", err)
	}
	<-done
}
