package assets

import (
	"fmt"
	"io/fs"
	"path"
	"strings"
)

// Load は指定されたファイルシステム(fileSystem)のディレクトリ(rootDir)から、 指定された接頭辞(prefix)を持つファイルを読み込み、マップとして返します。
func Load(fileSystem fs.FS, rootDir, prefix string) (map[string]string, error) {
	templates := make(map[string]string)

	entries, err := fs.ReadDir(fileSystem, rootDir)
	if err != nil {
		return nil, fmt.Errorf("ディレクトリ %s の読み込みに失敗: %w", rootDir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		modeName := strings.TrimPrefix(
			strings.TrimSuffix(fileName, path.Ext(fileName)),
			prefix,
		)

		filePath := path.Join(rootDir, fileName)
		content, err := fs.ReadFile(fileSystem, filePath)
		if err != nil {
			return nil, fmt.Errorf("ファイル %s の読み込みに失敗: %w", filePath, err)
		}

		templates[modeName] = string(content)
	}

	return templates, nil
}
