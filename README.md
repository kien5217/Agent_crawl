1) Điều kiện cần:
- phải go get các package trong go mod
- Phải sử dụng postgresql, bạn có thể thay đổi database_url trong config/config.yaml nếu cần
2) Cách chạy:
- Chạy migrate để tạo bảng: go run main.go migrate --config ./config/config.yaml
- Chạy schedule để enqueue các bài viết từ RSS và sitemap: go run main.go schedule --config ./config/config.yaml (Lưu ý: nếu bạn có nhiều nguồn với sitemap, có thể sẽ mất thời gian để enqueue hết, bạn có thể tạm thời disable sitemap trong sources.yaml để test nhanh)
- Chạy worker để bắt đầu crawl và phân loại: go run main.go worker --config ./config/config.yaml
- Để list topic cve mới nhất: go run main.go list cve 100(Đây là số lượng đường link mà bạn muốn liệt kê) --config ./config/config.yaml
