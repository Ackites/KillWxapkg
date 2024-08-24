package unpack

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"

	"github.com/Ackites/KillWxapkg/internal/key"

	"github.com/Ackites/KillWxapkg/internal/config"

	formatter2 "github.com/Ackites/KillWxapkg/internal/formatter"
)

type WxapkgFile struct {
	NameLen uint32
	Name    string
	Offset  uint32
	Size    uint32
}

// UnpackWxapkg 解包 wxapkg 文件并将内容保存到指定目录
func UnpackWxapkg(data []byte, outputDir string) ([]string, error) {
	reader := bytes.NewReader(data)

	// 读取文件头
	var firstMark byte
	if err := binary.Read(reader, binary.BigEndian, &firstMark); err != nil {
		return nil, fmt.Errorf("读取首标记失败: %v", err)
	}
	if firstMark != 0xBE {
		return nil, fmt.Errorf("无效的wxapkg文件: 首标记不正确")
	}

	var info1, indexInfoLength, bodyInfoLength uint32
	if err := binary.Read(reader, binary.BigEndian, &info1); err != nil {
		return nil, fmt.Errorf("读取info1失败: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &indexInfoLength); err != nil {
		return nil, fmt.Errorf("读取索引段长度失败: %v", err)
	}
	if err := binary.Read(reader, binary.BigEndian, &bodyInfoLength); err != nil {
		return nil, fmt.Errorf("读取数据段长度失败: %v", err)
	}

	// 验证长度的合理性
	totalLength := uint64(indexInfoLength) + uint64(bodyInfoLength)
	if totalLength > uint64(len(data)) {
		return nil, fmt.Errorf("文件长度不足, 文件损坏: 索引段(%d) + 数据段(%d) > 文件总长度(%d)", indexInfoLength, bodyInfoLength, len(data))
	}
	totalLength = uint64(len(data))

	var lastMark byte
	if err := binary.Read(reader, binary.BigEndian, &lastMark); err != nil {
		return nil, fmt.Errorf("读取尾标记失败: %v", err)
	}
	if lastMark != 0xED {
		return nil, fmt.Errorf("无效的wxapkg文件: 尾标记不正确")
	}

	var fileCount uint32
	if err := binary.Read(reader, binary.BigEndian, &fileCount); err != nil {
		return nil, fmt.Errorf("读取文件数量失败: %v", err)
	}

	// 计算索引段的预期结束位置
	expectedIndexEnd := uint64(reader.Size()) - uint64(bodyInfoLength)

	// 读取索引
	fileList := make([]WxapkgFile, fileCount)
	var filelistNames []string
	for i := range fileList {
		if err := binary.Read(reader, binary.BigEndian, &fileList[i].NameLen); err != nil {
			return nil, fmt.Errorf("读取文件名长度失败: %v", err)
		}

		if fileList[i].NameLen == 0 || fileList[i].NameLen > 1024 {
			return nil, fmt.Errorf("文件名长度 %d 不合理", fileList[i].NameLen)
		}

		nameBytes := make([]byte, fileList[i].NameLen)
		if _, err := io.ReadAtLeast(reader, nameBytes, int(fileList[i].NameLen)); err != nil {
			return nil, fmt.Errorf("读取文件名失败: %v", err)
		}

		fileList[i].Name = string(nameBytes)

		filelistNames = append(filelistNames, fileList[i].Name)

		if err := binary.Read(reader, binary.BigEndian, &fileList[i].Offset); err != nil {
			return nil, fmt.Errorf("读取文件偏移量失败: %v", err)
		}

		if err := binary.Read(reader, binary.BigEndian, &fileList[i].Size); err != nil {
			return nil, fmt.Errorf("读取文件大小失败: %v", err)
		}

		// 验证文件偏移量和大小
		fileEnd := uint64(fileList[i].Offset) + uint64(fileList[i].Size)
		if fileEnd > totalLength {
			return nil, fmt.Errorf("文件 %s 的结束位置 (%d) 超出了文件总长度 (%d)", fileList[i].Name, fileEnd, totalLength)
		}

		// 验证我们是否仍在索引段内
		currentPos := uint64(reader.Size()) - uint64(reader.Len())
		if currentPos > expectedIndexEnd {
			return nil, fmt.Errorf("索引读取超出预期范围: 当前位置 %d, 预期索引结束位置 %d", currentPos, expectedIndexEnd)
		}
	}

	// 验证是否正确读完了整个索引段
	currentPos := uint64(reader.Size()) - uint64(reader.Len())
	if currentPos != expectedIndexEnd {
		return nil, fmt.Errorf("索引段长度不符: 读取到位置 %d, 预期结束位置 %d", currentPos, expectedIndexEnd)
	}

	// 控制并发数
	const workerCount = 10
	var wg sync.WaitGroup
	fileChan := make(chan WxapkgFile, workerCount)
	errChan := make(chan error, workerCount)

	// 使用 sync.Pool 来复用缓冲区，减少内存分配和 GC 开销
	var bufferPool = sync.Pool{
		New: func() interface{} {
			return new(bytes.Buffer)
		},
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for file := range fileChan {
				if err := processFile(outputDir, file, reader, &bufferPool); err != nil {
					errChan <- fmt.Errorf("保存文件 %s 失败: %w", file.Name, err)
				}
			}
		}()
	}

	for _, file := range fileList {
		fileChan <- file
	}
	close(fileChan)

	// 等待所有 goroutine 完成
	wg.Wait()
	close(errChan)

	// 检查是否有错误
	if len(errChan) > 0 {
		return nil, <-errChan
	}

	//const configJSON = `{
	//    "description": "See https://developers.weixin.qq.com/miniprogram/dev/devtools/projectconfig.html",
	//    "setting": {
	//      "urlCheck": false
	//    }
	//  }`

	// 保存 project.private.config.json
	//configFile := filepath.Join(outputDir, "project.private.config.json")
	//if err := os.WriteFile(configFile, []byte(configJSON), 0755); err != nil {
	//	return nil, fmt.Errorf("保存文件 %s 失败: %w", configFile, err)
	//}

	return filelistNames, nil
}

// processFile 处理单个文件的读取、格式化和保存
func processFile(outputDir string, file WxapkgFile, reader io.ReaderAt, bufferPool *sync.Pool) error {
	fullPath := filepath.Join(outputDir, file.Name)
	dir := filepath.Dir(fullPath)

	// 创建目录
	if err := os.MkdirAll(dir, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 创建文件
	f, err := os.OpenFile(fullPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("创建文件失败: %w", err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			log.Printf("关闭文件 %s 失败: %v\n", file.Name, err)
		}
	}(f)

	// 使用 io.NewSectionReader 创建一个只读取指定部分的 Reader
	sectionReader := io.NewSectionReader(reader, int64(file.Offset), int64(file.Size))

	// 从 bufferPool 获取缓冲区
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()

	// 读取文件内容
	if _, err := io.Copy(buf, sectionReader); err != nil {
		return fmt.Errorf("读取文件内容失败: %w", err)
	}
	content := buf.Bytes()

	// 获取文件格式化器
	ext := filepath.Ext(file.Name)
	formatter, err := formatter2.GetFormatter(ext)
	if err == nil {
		content, err = formatter.Format(content)
		if err != nil {
			return fmt.Errorf("格式化文件失败: %w", err)
		}
	}

	// 写入文件内容
	if _, err := f.Write(content); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	configManager := config.NewSharedConfigManager()
	if sensitive, ok := configManager.Get("sensitive"); ok {
		if p, o := sensitive.(bool); o {
			if p {
				// 查找敏感信息
				if err := key.MatchRules(string(content)); err != nil {
					return fmt.Errorf("%v", err)
				}
			}
		}
	}

	return nil
}
