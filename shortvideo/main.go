package main

import (
	"fmt"
	"log"
	"net/http"
	"os/exec"
	
	"github.com/gin-gonic/gin"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

var minioClient *minio.Client

func initMinIO() {
	var err error
	minioClient, err = minio.New("localhost:9000", &minio.Options{
		Creds:  credentials.NewStaticV4("admin", "password", ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("MinIO连接失败: %v", err)
	}
}

func main() {
	initMinIO()
	
	r := gin.Default()
	
	r.POST("/upload", func(c *gin.Context) {
		file, err := c.FormFile("video")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "无效请求"})
			return
		}
		
		// 临时保存文件
		tempPath := "/tmp/" + file.Filename
		if err := c.SaveUploadedFile(file, tempPath); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "文件保存失败"})
			return
		}
		
		// 转码480p
		outputPath := "/tmp/480p_" + file.Filename
		cmd := exec.Command("ffmpeg", "-i", tempPath, "-vf", "scale=640:-1", outputPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "转码失败",
				"detail": string(output),
			})
			return
		}
		
		// 上传到MinIO
		_, err = minioClient.FPutObject(c, "videos", "480p_"+file.Filename, outputPath, minio.PutObjectOptions{})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "MinIO上传失败"})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{
			"message": "处理成功",
			"file": "480p_" + file.Filename,
			"url": "http://localhost:9000/videos/480p_" + file.Filename,
		})
	})
	
	fmt.Println("服务运行在: http://localhost:8080")
	r.Run(":8080")
}
