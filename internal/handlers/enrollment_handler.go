package handlers

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"smart_classroom/internal/db"
	"smart_classroom/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Recognition config (env-configurable). Defaults match the trained model in
// NhanDangMSSV/: cosine threshold 0.60, kNN k=5 weighted vote.
func tHigh() float64 { return envFloat("FACE_T_HIGH", 0.60) } // accept (auto-mark)
func tLow() float64  { return envFloat("FACE_T_LOW", 0.45) }  // below = unknown; between = review
func knnK() int {
	if v := os.Getenv("FACE_KNN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 5
}

func envFloat(key string, def float64) float64 {
	if v := os.Getenv(key); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return def
}

func formatVec(v []float64) string {
	parts := make([]string, len(v))
	for i, x := range v {
		parts[i] = strconv.FormatFloat(x, 'f', 6, 64)
	}
	return "[" + strings.Join(parts, ",") + "]"
}

func resolveStudent(studentID uint, mssv string) (models.Student, bool) {
	var s models.Student
	q := db.DB
	if studentID != 0 {
		q = q.Where("student_id = ?", studentID)
	} else {
		q = q.Where("mssv = ?", mssv)
	}
	if err := q.First(&s).Error; err != nil {
		return s, false
	}
	return s, true
}

// storeEmbeddings replaces (or appends) a student's reference embeddings.
func storeEmbeddings(s models.Student, embs [][]float64, source string, replace bool) error {
	if replace {
		if err := db.DB.Exec(`DELETE FROM face_embeddings WHERE student_id = ?`, s.StudentID).Error; err != nil {
			return err
		}
	}
	for _, e := range embs {
		if err := db.DB.Exec(
			`INSERT INTO face_embeddings (student_id, mssv, student_name, source, embedding) VALUES (?, ?, ?, ?, ?::vector)`,
			s.StudentID, s.MSSV, s.StudentName, source, formatVec(e)).Error; err != nil {
			return err
		}
	}
	return nil
}

// HandleEnrollFace stores a student's reference embedding(s). Accepts a single
// `embedding` or a list `embeddings` (original + augmented, like the FAISS
// gallery). `replace` (default true) clears the student's existing vectors first.
func HandleEnrollFace(c *gin.Context) {
	var req struct {
		StudentID  uint        `json:"student_id"`
		MSSV       string      `json:"mssv"`
		Embedding  []float64   `json:"embedding"`
		Embeddings [][]float64 `json:"embeddings"`
		Replace    *bool       `json:"replace"`
		Source     string      `json:"source"`
	}
	if err := c.BindJSON(&req); err != nil || (req.StudentID == 0 && req.MSSV == "") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "student_id or mssv required"})
		return
	}
	embs := req.Embeddings
	if len(req.Embedding) > 0 {
		embs = append(embs, req.Embedding)
	}
	if len(embs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "embedding(s) required"})
		return
	}
	for _, e := range embs {
		if len(e) != 512 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "each embedding must have 512 dims"})
			return
		}
	}
	s, ok := resolveStudent(req.StudentID, req.MSSV)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy học sinh"})
		return
	}
	replace := true
	if req.Replace != nil {
		replace = *req.Replace
	}
	source := req.Source
	if source == "" {
		source = "manual"
	}
	if err := storeEmbeddings(s, embs, source, replace); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không lưu được embedding"})
		return
	}
	var total int64
	db.DB.Raw(`SELECT count(*) FROM face_embeddings WHERE student_id = ?`, s.StudentID).Scan(&total)
	writeAudit(c, "enroll", "face", uintStr(s.StudentID), "Ghi danh khuôn mặt "+s.StudentName)
	c.JSON(http.StatusOK, gin.H{"message": "Đã ghi danh khuôn mặt", "student_id": s.StudentID, "samples": total})
}

// HandleEnrollPhoto accepts an image upload, sends it to the face-enroll service
// to extract embedding(s) with the same model, then stores them. Requires the
// optional FACE_ENROLL_URL service to be running.
func HandleEnrollPhoto(c *gin.Context) {
	svc := os.Getenv("FACE_ENROLL_URL")
	if svc == "" {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Dịch vụ trích xuất khuôn mặt chưa bật (FACE_ENROLL_URL). Xem docs/ENROLLMENT.md"})
		return
	}
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Thiếu ảnh (field 'image')"})
		return
	}
	studentID := parseUintParam(c.PostForm("student_id"))
	mssv := c.PostForm("mssv")
	s, ok := resolveStudent(studentID, mssv)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy học sinh"})
		return
	}

	// Forward the image to the embedding service.
	embs, faceCount, err := callEmbedService(svc, file)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Lỗi gọi dịch vụ trích xuất: " + err.Error()})
		return
	}
	if len(embs) == 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": "Không phát hiện khuôn mặt trong ảnh", "faces": faceCount})
		return
	}
	if err := storeEmbeddings(s, embs, "photo", true); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không lưu được embedding"})
		return
	}
	writeAudit(c, "enroll", "face", uintStr(s.StudentID), "Ghi danh khuôn mặt (ảnh) "+s.StudentName)
	c.JSON(http.StatusOK, gin.H{"message": "Đã ghi danh khuôn mặt từ ảnh", "student_id": s.StudentID, "samples": len(embs)})
}

// callEmbedService posts the uploaded image to FACE_ENROLL_URL/embed and returns
// the list of 512-d embeddings (original + augmented).
func callEmbedService(svc string, file *multipart.FileHeader) ([][]float64, int, error) {
	src, err := file.Open()
	if err != nil {
		return nil, 0, err
	}
	defer src.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("image", file.Filename)
	if err != nil {
		return nil, 0, err
	}
	if _, err := io.Copy(fw, src); err != nil {
		return nil, 0, err
	}
	w.Close()

	req, err := http.NewRequest("POST", strings.TrimRight(svc, "/")+"/embed", &buf)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	var out struct {
		Embeddings [][]float64 `json:"embeddings"`
		Faces      int         `json:"faces"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, 0, err
	}
	return out.Embeddings, out.Faces, nil
}

// HandleGetGallery returns ALL embeddings of students enrolled in a classroom,
// so a Jetson can rebuild a FAISS gallery and match on the edge (kNN vote).
func HandleGetGallery(c *gin.Context) {
	classroomID := parseUintParam(c.Query("classroom_id"))
	type row struct {
		StudentID   uint   `json:"student_id"`
		MSSV        string `json:"mssv"`
		StudentName string `json:"student_name"`
		Embedding   string `json:"embedding"`
	}
	var rows []row
	db.DB.Raw(`SELECT fe.student_id, fe.mssv, fe.student_name, fe.embedding::text AS embedding
	           FROM face_embeddings fe
	           WHERE fe.student_id IN (
	             SELECT DISTINCT cs.student_id FROM class_students cs
	             JOIN classes c ON c.class_id = cs.class_id WHERE c.classroom_id = ?)`, classroomID).Scan(&rows)
	c.JSON(http.StatusOK, gin.H{"classroom_id": classroomID, "count": len(rows), "faces": rows})
}

// HandleEnrollStatus lists students with how many face samples each has enrolled.
// Optional ?classroom_id= scopes to a room; ?q= filters by mssv/name; ?only=enrolled|missing.
func HandleEnrollStatus(c *gin.Context) {
	type row struct {
		StudentID   uint   `json:"student_id"`
		MSSV        string `json:"mssv"`
		StudentName string `json:"student_name"`
		Samples     int    `json:"samples"`
	}
	q := db.DB.Table("students s").
		Select(`s.student_id, s.mssv, s.student_name, COALESCE(cnt.n,0) AS samples`).
		Joins(`LEFT JOIN (SELECT student_id, count(*) n FROM face_embeddings GROUP BY student_id) cnt ON cnt.student_id = s.student_id`)
	if cid := parseUintParam(c.Query("classroom_id")); cid != 0 {
		q = q.Where(`s.student_id IN (SELECT DISTINCT cs.student_id FROM class_students cs JOIN classes c ON c.class_id=cs.class_id WHERE c.classroom_id = ?)`, cid)
	}
	if term := strings.TrimSpace(c.Query("q")); term != "" {
		like := "%" + term + "%"
		q = q.Where("s.mssv ILIKE ? OR s.student_name ILIKE ?", like, like)
	}
	switch c.Query("only") {
	case "enrolled":
		q = q.Where("COALESCE(cnt.n,0) > 0")
	case "missing":
		q = q.Where("COALESCE(cnt.n,0) = 0")
	}
	var rows []row
	q.Order("s.mssv").Limit(1000).Scan(&rows)
	var enrolled int64
	db.DB.Raw(`SELECT count(DISTINCT student_id) FROM face_embeddings`).Scan(&enrolled)
	c.JSON(http.StatusOK, gin.H{"students": rows, "enrolled_total": enrolled})
}

// HandleDeleteFace removes all reference embeddings for a student.
func HandleDeleteFace(c *gin.Context) {
	sid := parseUintParam(c.Param("student_id"))
	if sid == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "student_id required"})
		return
	}
	db.DB.Exec(`DELETE FROM face_embeddings WHERE student_id = ?`, sid)
	writeAudit(c, "delete", "face", uintStr(sid), "Xoá khuôn mặt đã ghi danh")
	c.JSON(http.StatusOK, gin.H{"message": "Đã xoá khuôn mặt đã ghi danh"})
}

// recognizeByEmbedding finds the best-matching student for an embedding using a
// kNN (k=FACE_KNN) weighted cosine vote within the classroom's gallery — the
// same scheme as the training notebook. confidence = sum(best votes)/k_valid.
func recognizeByEmbedding(classroomID uint, emb []float64) (uint, string, string, float64, bool) {
	vec := formatVec(emb)
	type nb struct {
		StudentID   uint
		MSSV        string
		StudentName string
		Sim         float64
	}
	var rows []nb
	err := db.DB.Raw(
		`SELECT fe.student_id, fe.mssv, fe.student_name, 1 - (fe.embedding <=> ?::vector) AS sim
		 FROM face_embeddings fe
		 WHERE fe.student_id IN (
		   SELECT DISTINCT cs.student_id FROM class_students cs
		   JOIN classes c ON c.class_id = cs.class_id WHERE c.classroom_id = ?)
		 ORDER BY fe.embedding <=> ?::vector LIMIT ?`, vec, classroomID, vec, knnK()).Scan(&rows).Error
	if err != nil || len(rows) == 0 {
		return 0, "", "", 0, false
	}
	type agg struct {
		sum        float64
		mssv, name string
	}
	votes := map[uint]*agg{}
	for _, r := range rows {
		a := votes[r.StudentID]
		if a == nil {
			a = &agg{mssv: r.MSSV, name: r.StudentName}
			votes[r.StudentID] = a
		}
		a.sum += r.Sim
	}
	var bestID uint
	var best *agg
	for id, a := range votes {
		if best == nil || a.sum > best.sum {
			best, bestID = a, id
		}
	}
	confidence := best.sum / float64(len(rows)) // average weighted vote (like notebook)
	return bestID, best.mssv, best.name, confidence, true
}

// HandleGetReviewQueue lists low-confidence recognitions (staff). A teacher only
// sees reviews for classrooms they are assigned to; admin sees all.
func HandleGetReviewQueue(c *gin.Context) {
	var rows []models.FaceReview
	q := db.DB.Order("created_at desc")
	if c.Query("status") != "" {
		q = q.Where("status = ?", c.Query("status"))
	} else {
		q = q.Where("status = ?", "pending")
	}
	if c.GetString("role") == "teacher" {
		ids, _ := scopedClassroomIDs(c)
		if len(ids) == 0 {
			c.JSON(http.StatusOK, []models.FaceReview{})
			return
		}
		q = q.Where("classroom_id IN ?", ids)
	}
	q.Limit(300).Find(&rows)
	c.JSON(http.StatusOK, rows)
}

// HandleReviewDecision confirms (→ creates attendance) or rejects a review item.
func HandleReviewDecision(c *gin.Context) {
	id := parseUintParam(c.Param("id"))
	var req struct {
		Decision string `json:"decision"` // confirm | reject
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	var fr models.FaceReview
	if err := db.DB.First(&fr, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy"})
		return
	}
	// A teacher may only review items for classrooms assigned to them.
	if !canManageClassroom(c, fr.ClassroomID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Bạn không được phân công lớp này"})
		return
	}
	if req.Decision == "confirm" {
		aid := uuid.New().String()
		cid := fr.ClassID
		subj := fr.Subject
		att := models.Attendance{
			ID: &aid, StudentID: fr.StudentID, ClassroomID: fr.ClassroomID, ClassID: &cid, Subject: &subj,
			Date: fr.Date, AttendanceStatus: models.StatusPresent, DetectionTime: fr.DetectionTime, DeviceID: fr.DeviceID,
		}
		db.DB.Where("student_id = ? AND class_id = ? AND date = ?", fr.StudentID, fr.ClassID, fr.Date).
			FirstOrCreate(&att)
		db.DB.Model(&fr).Update("status", "confirmed")
		writeAudit(c, "confirm", "face_review", uintStr(fr.ID), "Xác nhận điểm danh "+fr.StudentName)
		c.JSON(http.StatusOK, gin.H{"message": "Đã xác nhận điểm danh"})
		return
	}
	db.DB.Model(&fr).Update("status", "rejected")
	writeAudit(c, "reject", "face_review", uintStr(fr.ID), "Từ chối nhận diện "+fr.StudentName)
	c.JSON(http.StatusOK, gin.H{"message": "Đã từ chối"})
}
