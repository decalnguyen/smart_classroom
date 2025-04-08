package handlers

import (
	"net/http"

	"smart_classroom/db"
	"smart_classroom/models"

	"github.com/gin-gonic/gin"
)

func HandleGetBuildings(c *gin.Context) {
	var buildings []models.Building
	if err := db.DB.Find(&buildings).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve buildings"})
		return
	}
	c.JSON(http.StatusOK, buildings)
}

func HandlePostBuilding(c *gin.Context) {
	var building models.Building
	if err := c.BindJSON(&building); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if err := db.DB.Where("building_id = ?", building.BuildingID).First(&models.Building{}).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Building already exists"})
		return
	} else if err := db.DB.Create(&building).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create building"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Building created"})
}

func HandlePutBuilding(c *gin.Context) {
	id := c.Param("id")
	var building models.Building
	if err := db.DB.Where("building_id = ?", id).First(&building).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Building not found"})
		return
	}
	if err := c.BindJSON(&building); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if err := db.DB.Save(&building).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update building"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Building updated"})
}

func HandleDeleteBuilding(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Where("building_id = ?", id).Delete(&models.Building{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete building"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Building deleted"})
}

func HandleGetClassrooms(c *gin.Context) {
	var classrooms []models.Classroom
	if err := db.DB.Find(&classrooms).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve classrooms"})
		return
	}
	c.JSON(http.StatusOK, classrooms)
}

func HandlePostClassroom(c *gin.Context) {
	var classroom models.Classroom
	if err := c.BindJSON(&classroom); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if err := db.DB.Where("classroom_id = ?", classroom.ClassroomID).First(&models.Classroom{}).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Classroom already exists"})
		return
	} else if err := db.DB.Create(&classroom).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create classroom"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Classroom created"})
}

func HandlePutClassroom(c *gin.Context) {
	id := c.Param("id")
	var classroom models.Classroom
	if err := db.DB.Where("classroom_id = ?", id).First(&classroom).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Classroom not found"})
		return
	}
	if err := c.BindJSON(&classroom); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if err := db.DB.Save(&classroom).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update classroom"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Classroom updated"})
}

func HandleDeleteClassroom(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Where("classroom_id = ?", id).Delete(&models.Classroom{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete classroom"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Classroom deleted"})
}

func HandleGetStudents(c *gin.Context) {
	var students []models.Student
	if err := db.DB.Find(&students).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve students"})
		return
	}
	c.JSON(http.StatusOK, students)
}

func HandlePostStudent(c *gin.Context) {
	var student models.Student
	if err := c.BindJSON(&student); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if err := db.DB.Where("student_id = ?", student.StudentID).First(&models.Student{}).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Student already exists"})
		return
	} else if err := db.DB.Create(&student).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create student"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Student created"})
}

func HandlePutStudent(c *gin.Context) {
	id := c.Param("id")
	var student models.Student
	if err := db.DB.Where("student_id = ?", id).First(&student).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Student not found"})
		return
	}
	if err := c.BindJSON(&student); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if err := db.DB.Save(&student).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update student"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Student updated"})
}

func HandleDeleteStudent(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Where("student_id = ?", id).Delete(&models.Student{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete student"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Student deleted"})
}
func HandleGetTeachers(c *gin.Context) {
	var teachers []models.Teacher
	if err := db.DB.Find(&teachers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve teachers"})
		return
	}
	c.JSON(http.StatusOK, teachers)
}

func HandlePostTeacher(c *gin.Context) {
	var teacher models.Teacher
	if err := c.BindJSON(&teacher); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if err := db.DB.Where("teacher_id = ?", teacher.TeacherID).First(&models.Teacher{}).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Teacher already exists"})
		return
	} else if err := db.DB.Create(&teacher).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create teacher"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Teacher created"})
}

func HandlePutTeacher(c *gin.Context) {
	id := c.Param("id")
	var teacher models.Teacher
	if err := db.DB.Where("teacher_id = ?", id).First(&teacher).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Teacher not found"})
		return
	}
	if err := c.BindJSON(&teacher); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}
	if err := db.DB.Save(&teacher).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update teacher"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Teacher updated"})
}

func HandleDeleteTeacher(c *gin.Context) {
	id := c.Param("id")
	if err := db.DB.Where("teacher_id = ?", id).Delete(&models.Teacher{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete teacher"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Teacher deleted"})
}
