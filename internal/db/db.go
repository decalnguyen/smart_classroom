package db

import (
	"fmt"
	"log"
	"os"
	"time"

	"smart_classroom/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

// env returns the environment variable value or a fallback default.
func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// dsn builds the PostgreSQL connection string from environment variables,
// falling back to local-dev defaults so the stack still boots out of the box.
func dsn() string {
	return fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=%s",
		env("DB_HOST", "postgres"),
		env("DB_USER", "nhattoan"),
		env("DB_PASSWORD", "test123"),
		env("DB_NAME", "sensordata"),
		env("DB_PORT", "5432"),
		env("DB_SSLMODE", "disable"),
	)
}

func InitDB() {
	var err error

	// Postgres may still be starting up when this service boots, so retry the
	// connection for a while before giving up.
	for attempt := 1; attempt <= 30; attempt++ {
		DB, err = gorm.Open(postgres.Open(dsn()), &gorm.Config{})
		if err == nil {
			sqlDB, dbErr := DB.DB()
			if dbErr == nil && sqlDB.Ping() == nil {
				break
			}
			err = dbErr
		}
		log.Printf("Waiting for database (attempt %d/30): %v", attempt, err)
		time.Sleep(2 * time.Second)
	}
	if err != nil {
		log.Fatal("Failed to connect to database after retries:", err)
	}

	if err := DB.AutoMigrate(&models.User{}); err != nil {
		log.Fatal("Failed to migrate User:", err)
	}
	modelsToMigrate := []interface{}{
		&models.SenSorData{},
		&models.UserProfile{},
		&models.Face{},
		&models.Notification{},
		&models.Sensor{},
		&models.Building{},
		&models.Classroom{},
		&models.Student{},
		&models.Subject{},
		&models.Teacher{},
		&models.Attendance{},
		&models.ClassroomTeacher{},
		&models.Schedule{},
		&models.Electricity{},
		&models.Class{},
		&models.ClassStudent{},
		&models.Semester{},
		&models.Holiday{},
		&models.MakeupSession{},
		&models.LeaveRequest{},
		&models.AuditLog{},
		&models.DeviceCredential{},
		&models.FaceReview{},
	}
	if err := DB.AutoMigrate(modelsToMigrate...); err != nil {
		log.Fatal("Failed to migrate database models:", err)
	}
	runMigrations()
	log.Println("Database connection initialized and migrated successfully")
}

// runMigrations applies explicit, idempotent SQL that GORM AutoMigrate can't do:
// the pgvector extension (face embeddings) and a unique constraint that makes
// attendance idempotent at the DB layer (one row per student/class/date).
// NOTE: production should adopt a versioned migration tool (golang-migrate);
// this is the minimal explicit-SQL bootstrap.
func runMigrations() {
	stmts := []string{
		`CREATE EXTENSION IF NOT EXISTS vector`,
		// Migrate any old single-row-per-student gallery (PK student_id, no id column)
		// to the multi-embedding shape. Only drops when empty (gallery is rebuildable
		// from training data), so it is non-destructive for real data.
		`DO $$
		 BEGIN
		   IF EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name='face_embeddings')
		      AND EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='face_embeddings' AND column_name='student_id')
		      AND NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='face_embeddings' AND column_name='id')
		      AND (SELECT count(*) FROM face_embeddings) = 0 THEN
		     DROP TABLE face_embeddings;
		   END IF;
		 END $$`,
		// Face embeddings store (ArcFace 512-d). MULTIPLE reference vectors per
		// student (original + augmented), mirroring the trained FAISS gallery, so
		// recognition can do a kNN weighted vote like the training pipeline.
		`CREATE TABLE IF NOT EXISTS face_embeddings (id bigserial PRIMARY KEY, student_id bigint NOT NULL, mssv text, student_name text, source text, embedding vector(512), created_at timestamptz DEFAULT now())`,
		`CREATE INDEX IF NOT EXISTS idx_face_emb_student ON face_embeddings (student_id)`,
		// Idempotent attendance: one row per student/class/date.
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_attendance_student_class_date ON attendances (student_id, class_id, date)`,
		`CREATE INDEX IF NOT EXISTS idx_sensordata_ts ON sen_sor_data (timestamp)`,
	}
	for _, s := range stmts {
		if err := DB.Exec(s).Error; err != nil {
			log.Printf("Migration warning [%.40s...]: %v", s, err)
		}
	}
}
