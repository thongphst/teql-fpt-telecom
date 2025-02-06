package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strings"
	"net/http"

	_ "github.com/denisenkom/go-mssqldb"
	_ "github.com/go-sql-driver/mysql"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
)

var (
	currentDriver  string
	currentConnStr string
)

type Table struct {
	Name   string
	Storage string
	Port    string
}

func testConnection(driver, connStr string) error {
	db, err := sql.Open(driver, connStr)
	if err != nil {
		return fmt.Errorf("lỗi kết nối: %v", err)
	}
	defer db.Close()

	return db.Ping()
}

func executeQuery(query string) (string, error) {
	if currentConnStr == "" {
		return "", fmt.Errorf("chưa được kết nối database. Hãy /connect trước")
	}

	db, err := sql.Open(currentDriver, currentConnStr)
	if err != nil {
		return "", fmt.Errorf("lỗi kết nối: %v", err)
	}
	defer db.Close()

	rows, err := db.Query(query)
	if err != nil {
		return "", fmt.Errorf("truy vấn lỗi: %v", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return "", fmt.Errorf("cột bị lỗi: %v", err)
	}

	var result []map[string]interface{}
	for rows.Next() {
		vals := make([]interface{}, len(columns))
		valPtrs := make([]interface{}, len(columns))
		for i := range columns {
			valPtrs[i] = &vals[i]
		}

		if err := rows.Scan(valPtrs...); err != nil {
			return "", fmt.Errorf("scan lỗi: %v", err)
		}

		rowMap := make(map[string]interface{})
		for i, val := range vals {
			if val == nil {
				rowMap[columns[i]] = nil
			} else {
				rowMap[columns[i]] = val
			}
		}
		result = append(result, rowMap)
	}

	var sb strings.Builder
	for _, row := range result {
		for key, value := range row {
			sb.WriteString(fmt.Sprintf("〔%s〕%v\n", key, value))
		}
		sb.WriteString("──\n")
	}

	return sb.String(), nil
}

func main() {
	data4Search := Table{
		Name: "Sheet1",
		Port:  "Column_2",
	}

	var token = os.Getenv("TOKEN")
	if token == "" {
		log.Fatal("Thiếu bot token.")
	}

	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}

	bot.Debug = true
	log.Printf("authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	// Bind to the port set by Render
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // fallback port if PORT is not set
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Bot đang chạy ngon lành!")
	})

	go func() {
		log.Fatal(http.ListenAndServe(":"+port, nil))
	}()

	for update := range updates {
		if update.Message == nil {
			continue
		}

		if update.Message.IsCommand() {
			switch update.Message.Command() {
			case "start":
				helpMsg := tgbotapi.NewMessage(update.Message.Chat.ID,
					"*ᴛɪ̀ᴍ ᴋɪᴇ̂́ᴍ ᴛᴜ̉ ᴍᴀ̣ɴɢ* 🔎\n\n"+
						"`Các chức năng:`\n"+
						"*/connect*\n"+
						"*/search*\n"+
						"*/query* `〔lệnh truy vấn〕`")
				helpMsg.ParseMode = tgbotapi.ModeMarkdown
				bot.Send(helpMsg)

			case "connect":
				//msg := tgbotapi.NewMessage(update.Message.Chat.ID, "Hãy nhập chuỗi kết nối database.")
				//bot.Send(msg)

				//update = <-updates // Get the next update (user input)

				//connStr := update.Message.Text
				connStr :="postgresql://neondb_owner:npg_mNH1XnzoLTQ6@ep-raspy-shape-a8fyxfop-pooler.eastus2.azure.neon.tech/neondb?sslmode=require"
				if connStr == "" {
					errMsg := tgbotapi.NewMessage(update.Message.Chat.ID, "Chuỗi kết nối không hợp lệ.")
					bot.Send(errMsg)
					continue
				}

				var driver string
				switch {
				case strings.Contains(connStr, "postgresql://") || strings.Contains(connStr, "postgres://"):
					driver = "postgres"
				case strings.Contains(connStr, "@tcp("):
					driver = "mysql"
				case strings.Contains(connStr, "sqlserver://") || strings.Contains(connStr, "server="):
					driver = "sqlserver"
				default:
					errMsg := tgbotapi.NewMessage(update.Message.Chat.ID, "Database này không được hỗ trợ.")
					bot.Send(errMsg)
					continue
				}

				err := testConnection(driver, connStr)
				if err != nil {
					errMsg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Lỗi kết nối: %v", err))
					bot.Send(errMsg)
					continue
				}

				currentDriver = driver
				currentConnStr = connStr

				msg = tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Đã kết nối: %s", strings.ToLower(driver)))
				bot.Send(msg)

			case "query":
				query := update.Message.CommandArguments()
				if query == "" {
					errMsg := tgbotapi.NewMessage(update.Message.Chat.ID,
						"Lỗi: /query 〔lệnh truy vấn〕.")
					bot.Send(errMsg)
					continue
				}

				result, err := executeQuery(query)
				if err != nil {
					errMsg := tgbotapi.NewMessage(update.Message.Chat.ID,
						fmt.Sprintf("Lỗi: %v", err))
					bot.Send(errMsg)
					continue
				}

				msg := tgbotapi.NewMessage(update.Message.Chat.ID, result)
				bot.Send(msg)
				
			case "search":
				// Ask for Port Pon
				askMsg := tgbotapi.NewMessage(update.Message.Chat.ID, "Nhập Port Pon của tủ bạn tìm kiếm.")
				bot.Send(askMsg)

				// Wait for the user's response
				update = <-updates // Get the next update (user input)

				portPon := update.Message.Text

				// Construct the query with LIKE to find the Port Pon
				query := fmt.Sprintf("SELECT * FROM %s WHERE %s LIKE '%%%s%%'", data4Search.Name, data4Search.Port, portPon)

				// Execute the query
				result, err := executeQuery(query)
				if err != nil {
					errMsg := tgbotapi.NewMessage(update.Message.Chat.ID, fmt.Sprintf("Lỗi: %v", err))
					bot.Send(errMsg)
					continue
				}

				// Send the result back to Telegram
				if result == "" {
					result = "Không tìm thấy kết quả."
				}
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, result)
				bot.Send(msg)

			default:
				helpMsg := tgbotapi.NewMessage(update.Message.Chat.ID,
					"Lỗi, sài /start để biết các chức năng.")
				bot.Send(helpMsg)
			}
		}
	}
}
