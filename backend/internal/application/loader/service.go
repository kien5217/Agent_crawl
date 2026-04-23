package loader

import (
	config "Agent_Crawl/internal/domain/config"
	"errors"
	"os"

	"gopkg.in/yaml.v3"
)

// hàm này chịu trách nhiệm tải cấu hình ứng dụng từ các tệp YAML. Nó đọc cấu hình chính từ configPath, danh sách chủ đề từ topicsPath và danh sách nguồn từ sourcesPath. Nếu có lỗi trong quá trình đọc hoặc phân tích cú pháp của bất kỳ tệp nào, nó sẽ trả về lỗi. Nếu tất cả các tệp được tải thành công, nó sẽ trả về một struct AppConfig chứa tất cả thông tin cấu hình cần thiết cho ứng dụng.
func LoadAll(configPath, topicsPath, sourcesPath string) (*config.AppConfig, error) {
	var cfg config.Config
	if err := loadYAML(configPath, &cfg); err != nil {
		return nil, err
	}

	var topics config.TopicsFile
	if err := loadYAML(topicsPath, &topics); err != nil {
		return nil, err
	}

	var sources config.SourcesFile
	if err := loadYAML(sourcesPath, &sources); err != nil {
		return nil, err
	}

	if cfg.DatabaseURL == "" {
		return nil, errors.New("database_url is empty (use ${DATABASE_URL} or a literal)")
	}
	return &config.AppConfig{Config: cfg, Topics: topics, Sources: sources}, nil
}

// hàm tiện ích để tải một tệp YAML vào một struct Go. Nó đọc nội dung của tệp tại đường dẫn được chỉ định và sử dụng yaml.Unmarshal để phân tích cú pháp nội dung thành struct được cung cấp. Nếu có lỗi trong quá trình đọc tệp hoặc phân tích cú pháp, nó sẽ trả về lỗi đó.
func loadYAML(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(b, out)
}
