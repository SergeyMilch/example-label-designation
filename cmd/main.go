package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

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
			"direction": "DESC"
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
	URL := os.Getenv("REQ_URL_FACTS")

	// Подготовка данных для отправки в формате form data
	data := url.Values{}
	data.Set("period_start", eventData.Params.Period.Start)
	data.Set("period_end", eventData.Params.Period.End)
	data.Set("period_key", eventData.Params.Period.TypeKey)
	// data.Set("indicator_to_mo_id", strconv.Itoa(eventData.Params.IndicatorToMoID))
	data.Set("indicator_to_mo_id", "315914")
	data.Set("value", "1")
	data.Set("is_plan", "0")
	data.Set("auth_user_id", "40")

	// Преобразование формата даты
	formattedTime, err := time.Parse(time.RFC3339, eventData.Time)
	if err != nil {
		return 0, err
	}
	data.Set("fact_time", formattedTime.Format("2006-01-02"))

	// Формирование supertags в формате JSON
	supertags := fmt.Sprintf(`[{"tag":{"id": %d, "name": "КТО", "key": "Who", "values_source": 0}, "value": "%s"}]`, eventData.Author.UserID, eventData.Author.UserName)
	data.Set("supertags", supertags)

	// comment := fmt.Sprintf(`[{"indicator_to_mo_id": %d, "platform": "%s"}]`, eventData.Params.IndicatorToMoID, eventData.Params.Platform)
	comment := fmt.Sprintf(`[{"indicator_to_mo_id": %s, "platform": "%s"}]`, "315914", eventData.Params.Platform)
	data.Set("comment", comment)

	// fmt.Println(data)

	// Отключение проверки SSL
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{Transport: tr}

	// Создание запроса
	req, err := http.NewRequest("POST", URL, strings.NewReader(data.Encode()))
	if err != nil {
		return 0, err
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	// Добавляем куку
	req.Header.Add("Cookie", os.Getenv("COOKIE_SESS"))

	// Отправка запроса
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		fmt.Println("Ошибка при запросе. Код:", resp.StatusCode, "Тело:", string(body))
		return 0, fmt.Errorf("Ошибка при запросе. Код: %d", resp.StatusCode)
	}

	// fmt.Println("Ответ сервера:", resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	// log.Println("Ответ сервера:", string(body))

	var response map[string]interface{}
	err = json.Unmarshal(body, &response)
	if err != nil {
		return 0, err
	}

	indicatorToMoFactID, ok := response["DATA"].(map[string]interface{})["indicator_to_mo_fact_id"].(float64)
	if !ok {
		return 0, fmt.Errorf("Не удалось получить indicator_to_mo_fact_id")
	}

	// log.Println("indicator_to_mo_fact_id:", indicatorToMoFactID)

	return int(indicatorToMoFactID), nil
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
