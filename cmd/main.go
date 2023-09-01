package main

import (
	"context"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

const (
	region         string = "us-east-1"
	avaBucket      string = "keruenava"
	documentBucket string = "keruendoc"
	accessKey             = "O4819_admin"
	secretKey             = "tAJNrfSh7pkX"
)

var (
	endpoint = "https://storage.oblako.kz:443"
)

// Danila not this file to run !!!

func main() {
	// Disable certificate verification
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	// Create an AWS session
	sess, err := session.NewSession(&aws.Config{
		Region:           aws.String(region),
		Endpoint:         aws.String(endpoint),
		Credentials:      credentials.NewStaticCredentials(accessKey, secretKey, ""),
		S3ForcePathStyle: aws.Bool(true),
	})
	if err != nil {
		log.Println("Failed to create session:", err)
		return
	}

	s3Client := s3.New(sess)

	router := gin.Default()
	router.Use(gin.Recovery())
	router.MaxMultipartMemory = 32 << 20 // 32 Mb

	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"PUT", "PATCH", "POST", "GET", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Length", "Authorization", "X-CSRF-Token", "Content-Type", "Accept", "X-Requested-With", "Bearer", "Authority"},
		ExposeHeaders:    []string{"Content-Length", "Authorization", "Content-Type", "application/json", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Accept", "Origin", "Cache-Control", "X-Requested-With"},
		AllowCredentials: true,
		AllowOriginFunc: func(origin string) bool {
			return origin == "https://s3.qkeruen.kz"
		},
	}))

	router.POST("/ava/upload/:filename", func(c *gin.Context) {
		go func(ctx *gin.Context) {
			fileName := c.Param("filename")
			// Retrieve the uploaded file from the request
			file, err := c.FormFile("file")
			if err != nil {
				log.Println("Failed to retrieve file:", err)
				return
			}

			// Open the uploaded file
			fileData, err := file.Open()
			if err != nil {
				log.Println("Failed to open file:", err)
				return
			}
			defer fileData.Close()

			// Create a unique key for the uploaded file in S3
			key := fileName

			// Upload the file to S3
			_, err = s3Client.PutObjectWithContext(ctx, &s3.PutObjectInput{
				Bucket: aws.String(avaBucket),
				Key:    aws.String(key),
				Body:   fileData,
			})
			if err != nil {
				log.Println("Failed to upload file to S3:", err)
				return
			}

			log.Println("Avatar uploaded successfully")
		}(c.Copy())

		c.String(http.StatusOK, "File upload in progress")
	})

	router.POST("/doc/upload/:filename", func(c *gin.Context) {
		go func(ctx *gin.Context) {
			fileName := c.Param("filename")
			// Retrieve the uploaded file from the request
			file, err := c.FormFile("file")
			if err != nil {
				log.Println("Failed to retrieve file:", err)
				return
			}

			// Open the uploaded file
			fileData, err := file.Open()
			if err != nil {
				log.Println("Failed to open file:", err)
				return
			}
			defer fileData.Close()

			// Create a unique key for the uploaded file in S3
			key := fileName

			// Upload the file to S3
			_, err = s3Client.PutObjectWithContext(ctx, &s3.PutObjectInput{
				Bucket: aws.String(documentBucket),
				Key:    aws.String(key),
				Body:   fileData,
			})
			if err != nil {
				log.Println("Failed to upload file to S3:", err)
				return
			}

			log.Println("Document uploaded successfully")
		}(c.Copy())

		c.String(http.StatusOK, "File upload in progress")
	})

	router.POST("/ava/download/:filename", func(c *gin.Context) {
		filename := c.Param("filename")

		go func(ctx *gin.Context) {
			// Download the file from S3
			resp, err := s3Client.GetObjectWithContext(ctx, &s3.GetObjectInput{
				Bucket: aws.String(avaBucket),
				Key:    aws.String(filename),
			})
			if err != nil {
				log.Println("Failed to download file from S3:", err)
				return
			}
			defer resp.Body.Close()

			// Set the appropriate headers
			c.Header("Content-Disposition", "attachment; filename="+filename)
			c.Header("Content-Type", "application/octet-stream")
			c.Header("Content-Transfer-Encoding", "binary")

			// Stream the file directly to the response
			if _, err = io.Copy(c.Writer, resp.Body); err != nil {
				log.Println("Failed to stream file:", err)
				return
			}
		}(c.Copy())

		c.String(http.StatusOK, "File download in progress")
	})

	router.POST("/doc/download/:filename", func(c *gin.Context) {
		filename := c.Param("filename")

		go func(ctx *gin.Context) {
			// Download the file from S3
			resp, err := s3Client.GetObjectWithContext(ctx, &s3.GetObjectInput{
				Bucket: aws.String(documentBucket),
				Key:    aws.String(filename),
			})
			if err != nil {
				log.Println("Failed to download file from S3:", err)
				return
			}
			defer resp.Body.Close()

			// Set the appropriate headers
			c.Header("Content-Disposition", "attachment; filename="+filename)
			c.Header("Content-Type", "application/octet-stream")
			c.Header("Content-Transfer-Encoding", "binary")

			// Stream the file directly to the response
			if _, err = io.Copy(c.Writer, resp.Body); err != nil {
				log.Println("Failed to stream file:", err)
				return
			}
		}(c.Copy())

		c.String(http.StatusOK, "File download in progress")
	})

	// Create a server with graceful shutdown
	srv := &http.Server{
		Addr:    ":3001",
		Handler: router,
	}

	// Start the server in a separate goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Server failed to start:", err)
		}
	}()

	// Wait for interrupt signal to gracefully shut down the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	// Set a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Shutdown the server
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server shutdown error:", err)
	}

	log.Println("Server stopped")

}
