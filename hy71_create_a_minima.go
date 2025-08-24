package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"net/http"

	_ "github.com/lib/pq" // Import PostgreSQL driver
)

// Config holds the application configuration
type Config struct {
	DBUsername string
	DBPassword string
	DBName     string
}

// DataSource holds the data source information
type DataSource struct {
	Name        string
	Description string
	Data        []map[string]string
}

// Dashboard holds the dashboard data
type Dashboard struct {
	Title       string
	DataSources []DataSource
}

func main() {
	// Load configuration from environment variables
	cfg := Config{
		DBUsername: "your_username",
		DBPassword: "your_password",
		DBName:     "your_database",
	}

	// Establish database connection
	db, err := sql.Open("postgres", fmt.Sprintf("user=%s password=%s dbname=%s sslmode=disable", cfg.DBUsername, cfg.DBPassword, cfg.DBName))
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	// Create a dashboard
	dashboard := Dashboard{
		Title: "Minimalist Data Pipeline Dashboard",
	}

	// Query data sources from database
	rows, err := db.Query("SELECT name, description FROM data_sources")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer rows.Close()

	dataSources := make([]DataSource, 0)
	for rows.Next() {
		var name, description string
		err = rows.Scan(&name, &description)
		if err != nil {
			fmt.Println(err)
			return
		}
		dataSource := DataSource{
			Name:        name,
			Description: description,
			Data:        make([]map[string]string, 0),
		}
		dataSources = append(dataSources, dataSource)
	}

	// Query data for each data source
	for i, ds := range dataSources {
		rows, err = db.Query(fmt.Sprintf("SELECT * FROM %s", ds.Name))
		if err != nil {
			fmt.Println(err)
			return
		}
		defer rows.Close()

		for rows.Next() {
			columns, err := rows.Columns()
			if err != nil {
				fmt.Println(err)
				return
			}
			values := make([]sql.RawBytes, len(columns))
			scanArgs := make([]interface{}, len(values))
			for i := range values {
				scanArgs[i] = &values[i]
			}
			err = rows.Scan(scanArgs...)
			if err != nil {
				fmt.Println(err)
				return
			}
			rowData := make(map[string]string)
			for j, col := range columns {
				rowData[col] = string(values[j])
			}
			dataSources[i].Data = append(dataSources[i].Data, rowData)
		}
	}

	dashboard.DataSources = dataSources

	// Create an HTTP server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	 tmpl, err := template.New("dashboard").Parse(`
            <html>
            <head>
                <title>{{.Title}}</title>
            </head>
            <body>
                <h1>{{.Title}}</h1>
                <ul>
                {{range .DataSources}}
                    <li>
                        <h2>{{.Name}}</h2>
                        <p>{{.Description}}</p>
                        <table>
                            <tr>
                                {{range $i, $col := .Data}}
                                    <th>{{$col}}</th>
                                {{end}}
                            </tr>
                            {{range .Data}}
                                <tr>
                                    {{range $i, $val := .}}
                                        <td>{{$val}}</td>
                                    {{end}}
                                </tr>
                            {{end}}
                        </table>
                    </li>
                {{end}}
                </ul>
            </body>
            </html>
        `)
		if err != nil {
			fmt.Println(err)
			return
		}
		err = tmpl.Execute(w, dashboard)
		if err != nil {
			fmt.Println(err)
			return
		}
	})

	http.ListenAndServe(":8080", nil)
}