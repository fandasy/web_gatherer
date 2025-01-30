package psql

import (
	"context"
	"database/sql"
	"fmt"
	"project/internal/models"
	"testing"
	"time"

	"project/internal/config"

	_ "github.com/lib/pq"
)

var s *Storage

func init() {
	cfg := &config.DB{
		DBHost:     "localhost",
		DBPort:     5432,
		DBName:     "testdb",
		DBUser:     "username",
		DBPassword: "password",
	}

	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	if err := db.Ping(); err != nil {
		panic(err)
	}

	s = &Storage{
		db: db,
	}
}

func TestInsertAndGetWebMessages(t *testing.T) {
	ctx := context.Background()

	// Тестовые сообщения
	testMessages := []models.WebMessage{
		{GroupName: "group1", Text: "Test message 1", TopicName: "topic1", Metadata: []models.MetaPair{{Url: "1", Type: "Photo"}}, CreatedAt: time.Now().Add(1 * time.Minute)},
		{GroupName: "group2", Text: "Test message 2", TopicName: "topic2", Metadata: nil, CreatedAt: time.Now().Add(2 * time.Minute)},
		{GroupName: "group3", Text: "Test message 3", TopicName: "topic3", Metadata: nil, CreatedAt: time.Now().Add(3 * time.Minute)},
		{GroupName: "group4", Text: "Test message 4", TopicName: "topic4", Metadata: nil, CreatedAt: time.Now().Add(4 * time.Minute)},
		{GroupName: "group5", Text: "Test message 5", TopicName: "topic5", Metadata: nil, CreatedAt: time.Now().Add(5 * time.Minute)},
	}

	// Вставка сообщений
	err := s.InsertWebMessages(ctx, testMessages)
	if err != nil {
		t.Fatalf("Failed to insert messages: %v", err)
	}

	// Получение сообщений
	messages, err := s.GetWebMessages(ctx, 0)
	if err != nil {
		t.Fatalf("Failed to get messages: %v", err)
	}

	// Проверка содержимого сообщений
	for _, msg := range messages {
		fmt.Println(msg)
	}
}
