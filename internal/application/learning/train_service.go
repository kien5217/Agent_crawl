package learning

import (
	"encoding/json"

	"Agent_Crawl/internal/domain/model"
	ml "Agent_Crawl/internal/infrastructure/machine_learning"
)

type TrainResult struct {
	NumSamples int
	NumClasses int
	VocabSize  int
}

// TrainFromDocs thực hiện việc huấn luyện một mô hình học máy từ một tập hợp các tài liệu đã được gán nhãn. Nó nhận vào một slice các DocForLearning, một slice các tên lớp và một giá trị minDF để xây dựng vectorizer TF-IDF. Hàm này sẽ tạo ra một vectorizer TF-IDF dựa trên văn bản của các tài liệu, sau đó biến đổi văn bản thành các vector đặc trưng và huấn luyện một mô hình logistic regression với các vector này và nhãn tương ứng. Kết quả trả về là một Bundle chứa vectorizer và mô hình đã được huấn luyện, cùng với một TrainResult chứa thông tin về số lượng mẫu, số lượng lớp và kích thước từ vựng.
func TrainFromDocs(train []model.LearningDocument, classes []string, minDF int) (*ml.Bundle, *TrainResult) {
	// map topic -> class index
	classIndex := map[string]int{} // topicID -> class index
	for i, c := range classes {
		classIndex[c] = i
	}

	texts := make([]string, 0, len(train)) // title + content
	ys := make([]int, 0, len(train))
	for _, d := range train {
		ci, ok := classIndex[d.TopicID]
		if !ok {
			continue
		}
		texts = append(texts, d.Title+"\n"+d.Content)
		ys = append(ys, ci)
	}

	vec := ml.New(minDF) // minDF giúp loại bỏ các từ quá phổ biến, có thể không mang nhiều thông tin phân biệt giữa các lớp. Bạn có thể điều chỉnh giá trị này dựa trên kích thước tập dữ liệu và độ đa dạng của văn bản.
	vec.Fit(texts)

	xs := make([]ml.SparseVector, len(texts)) // biến đổi văn bản thành vector đặc trưng
	for i, t := range texts {
		xs[i] = vec.Transform(t)
	}

	m := ml.NewModel(len(classes), len(vec.IDF), classes)
	m.Lambda = 1e-4
	m.TrainSGD(xs, ys, ml.TrainOptions{Epochs: 6, LR: 0.2, Shuffle: true}) // Bạn có thể điều chỉnh các siêu tham số này (số epoch, learning rate) dựa trên kích thước tập dữ liệu và độ phức tạp của mô hình. Việc shuffle dữ liệu trước mỗi epoch cũng giúp cải thiện hiệu suất huấn luyện.

	return &ml.Bundle{Vectorizer: vec, Model: m}, &TrainResult{
		NumSamples: len(xs),
		NumClasses: len(classes),
		VocabSize:  len(vec.IDF),
	}
}

// ClassesJSON là một hàm tiện ích để chuyển đổi slice các tên lớp thành một chuỗi JSON. Hàm này sử dụng json.Marshal để mã hóa slice thành định dạng JSON và trả về kết quả dưới dạng byte slice. Nếu có lỗi trong quá trình mã hóa, nó sẽ trả về một byte slice rỗng.
func ClassesJSON(classes []string) []byte {
	b, _ := json.Marshal(classes)
	return b
}
