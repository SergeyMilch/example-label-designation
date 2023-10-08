package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/SergeyMilch/example-label-designation/models"

	"github.com/joho/godotenv"
)

func makeRequestWithCookie() ([]models.EventData, error) {
	url := os.Getenv("REQ_URL_EVENTS")

	// Отключаем проверку SSL
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	// Подготовка запроса
	payload := strings.NewReader(`{
		"filter": {
			"field": {
				"key": "type",
				"sign": "LIKE",
				"values": [
					"MATRIX_REQUEST"
				]
			}
		},
		"sort": {
			"fields": [
				"time"
			],
			"direction": "desc"
		},
		"limit": 10
	}`)

	req, err := http.NewRequest("GET", url, payload)
	if err != nil {
		return nil, err
	}

	req.Header.Add("Content-Type", "application/json")

	// Добавляем куку
	req.Header.Add("Cookie", os.Getenv("COOKIE_SESS"))

	// Отправка запроса
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	// Чтение ответа
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Обработка ответа
	var response struct {
		Data struct {
			Rows []models.EventData `json:"rows"`
		} `json:"DATA"`
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, err
	}

	// Извлечение и обработка данных
	var jsonData []map[string]interface{}
	for _, row := range response.Data.Rows {
		data := map[string]interface{}{
			"Group":   row.Group,
			"Type":    row.Type,
			"Message": row.Msg,
			"Author": map[string]interface{}{
				"UserID":   row.Author.UserID,
				"UserName": row.Author.UserName,
				"MoID":     row.Author.MoID,
			},
			"Params": map[string]interface{}{
				"IndicatorToMoID": row.Params.IndicatorToMoID,
				"Platform":        row.Params.Platform,
			},
			"Period": map[string]interface{}{
				"Start":   row.Params.Period.Start,
				"End":     row.Params.Period.End,
				"TypeID":  row.Params.Period.TypeID,
				"TypeKey": row.Params.Period.TypeKey,
			},
		}
		jsonData = append(jsonData, data)
	}

	// Вывод данных в формате JSON
	jsonOutput, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return nil, err
	}

	fmt.Println(string(jsonOutput))

	return response.Data.Rows, nil
}

func saveFactToMySQL(eventData models.EventData) (int, error) {
	url := os.Getenv("REQ_URL_FACTS")

	// Отключаем проверку SSL
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	// Данные для запроса
	supertags := fmt.Sprintf(`[{"tag":{"id": %d, "name": "КТО", "key": "Who", "values_source": 0}, "value": "%s"}]`, eventData.Author.UserID, eventData.Author.UserName)

	comment := fmt.Sprintf(`[{"indicator_to_mo_id": %d, "platform": "%s"}]`, eventData.Params.IndicatorToMoID, eventData.Params.Platform)

	payload := fmt.Sprintf(`{
		"period_start": "%s",
		"period_end": "%s",
		"period_key": "%s",
		"indicator_to_mo_id": %d,
		"value": %d,
		"fact_time": "%s",
		"is_plan": %d,
		"supertags": %s,
		"auth_user_id": %d,
		"comment": "%s"
	}`, eventData.Params.Period.Start, eventData.Params.Period.End, eventData.Params.Period.TypeKey,
		eventData.Params.IndicatorToMoID, 1, eventData.Time, 0,
		supertags,
		40, comment)

	req, err := http.NewRequest("POST", url, strings.NewReader(payload))
	if err != nil {
		return 0, err
	}

	// fmt.Println(payload)

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Cookie", os.Getenv("COOKIE_SESS"))

	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	// if resp.StatusCode != http.StatusOK {
	// 	body, _ := io.ReadAll(resp.Body)
	// 	fmt.Println("Ошибка при запросе. Код:", resp.StatusCode, "Тело:", string(body))
	// 	return 0, fmt.Errorf("Ошибка при запросе. Код: %d", resp.StatusCode)
	// }

	// Обработка ответа
	var mysqlResp models.MySQLResponse
	err = json.NewDecoder(resp.Body).Decode(&mysqlResp)
	if err != nil {
		return 0, err
	}

	// Возвращаем indicator_to_mo_fact_id
	return mysqlResp.IndicatorToMoFactID, nil
}

func loadEnvVariables() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
}

func main() {

	// Загружаем переменные окружения
	loadEnvVariables()

	eventData, err := makeRequestWithCookie()
	if err != nil {
		fmt.Println("Ошибка при отправке запроса:", err)
		return
	}

	for _, data := range eventData {
		factID, err := saveFactToMySQL(data)
		if err != nil {
			fmt.Println("Ошибка при сохранении факта:", err)
			continue
		}

		fmt.Printf("indicator_to_mo_fact_id: %d\n", factID)
	}
}
