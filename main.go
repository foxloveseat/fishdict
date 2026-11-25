package main

import (
	"database/sql"
	"fishweb/deftypes"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	_ "modernc.org/sqlite"
)

type PageData struct {
	Trips  []deftypes.Trip
	Users  []string
	Fishes []string
	Lures  []string
}

func main() {
	db, err := sql.Open("sqlite", "./fishing.db")
	if err != nil {
		log.Fatalf("Error opening DB: %v", err)
	}
	defer db.Close()

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS trips (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date TEXT,
		place TEXT,
		user_name TEXT
	);`)
	if err != nil {
		log.Fatalf("Ошибка создания trips: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS lures (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE
	);`)
	if err != nil {
		log.Fatalf("Ошибка создания lures: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE
	);`)
	if err != nil {
		log.Fatalf("Ошибка создания users: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS fishes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE
	);`)
	if err != nil {
		log.Fatalf("Ошибка создания fishes: %v", err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS catches (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		trip_id INTEGER,
		fish_name TEXT,
		count INTEGER,
		lure_name TEXT,
		FOREIGN KEY (trip_id) REFERENCES trips(id)
	);`)
	if err != nil {
		log.Fatalf("Ошибка создания catches: %v", err)
	}

	r := gin.Default()
	r.LoadHTMLGlob("templates/*")

	r.GET("/", func(c *gin.Context) {
		rows, err := db.Query("SELECT id, date, place, user_name FROM trips ORDER BY id DESC")
		if err != nil {
			c.String(http.StatusInternalServerError, "ОШИБКА ЧТЕНИЯ БАЗЫ")
			return
		}
		defer rows.Close()

		var trips []deftypes.Trip
		for rows.Next() {
			var t deftypes.Trip
			var userName string
			if err := rows.Scan(&t.Id, &t.Date, &t.Place, &userName); err != nil {
				continue
			}
			t.User = deftypes.User{Name: userName}

			catchRows, err := db.Query("SELECT fish_name, count, lure_name FROM catches WHERE trip_id = ?", t.Id)
			if err == nil {
				for catchRows.Next() {
					var cat deftypes.Catch
					if err := catchRows.Scan(&cat.Fish.Name, &cat.Count, &cat.Lure.Name); err == nil {
						t.Catches = append(t.Catches, cat)
					}
				}
				catchRows.Close()
			}
			trips = append(trips, t)
		}

		var users []string
		userRows, err := db.Query("SELECT name FROM users")
		if err == nil {
			for userRows.Next() {
				var name string
				if err := userRows.Scan(&name); err == nil {
					users = append(users, name)
				}
			}
			userRows.Close()
		}

		var fishes []string
		fishRows, err := db.Query("SELECT name FROM fishes")
		if err == nil {
			for fishRows.Next() {
				var name string
				if err := fishRows.Scan(&name); err == nil {
					fishes = append(fishes, name)
				}
			}
			fishRows.Close()
		}

		var lures []string
		lureRows, err := db.Query("SELECT name FROM lures")
		if err == nil {
			for lureRows.Next() {
				var name string
				if err := lureRows.Scan(&name); err == nil {
					lures = append(lures, name)
				}
			}
			lureRows.Close()
		}

		data := PageData{
			Trips:  trips,
			Users:  users,
			Fishes: fishes,
			Lures:  lures,
		}

		c.HTML(http.StatusOK, "index.html", data)
	})

	r.POST("/add", func(c *gin.Context) {
		date := c.PostForm("date")
		place := c.PostForm("place")
		userName := c.PostForm("user_name")

		fishNames := c.PostFormArray("fish_name[]")
		counts := c.PostFormArray("count[]")
		lureNames := c.PostFormArray("lure_name[]")

		var catches []deftypes.Catch
		for i := 0; i < len(fishNames); i++ {
			cCount := 1
			if i < len(counts) {
				if parsed, err := strconv.Atoi(counts[i]); err == nil {
					cCount = parsed
				}
			}

			lureName := ""
			if i < len(lureNames) {
				lureName = lureNames[i]
			}

			catches = append(catches, deftypes.Catch{
				Fish:  deftypes.Fish{Name: fishNames[i]},
				Count: cCount,
				Lure:  deftypes.Lure{Name: lureName},
			})
		}

		newTrip := deftypes.Trip{
			Date:    date,
			Place:   place,
			User:    deftypes.User{Name: userName},
			Catches: catches,
		}

		err = SaveTrip(db, newTrip)
		if err != nil {
			c.String(http.StatusInternalServerError, "Не удалось сохранить рыбалку: %v", err)
			return
		}

		c.Redirect(http.StatusSeeOther, "/")
	})

	r.POST("/delete/:id", func(c *gin.Context) {
		idStr := c.Param("id")
		tripID, err := strconv.Atoi(idStr)
		if err != nil {
			c.String(http.StatusBadRequest, "Неверный ID")
			return
		}

		err = DeleteTrip(db, tripID)
		if err != nil {
			c.String(http.StatusInternalServerError, "Не удалось удалить: %v", err)
			return
		}

		c.Redirect(http.StatusSeeOther, "/")
	})

	log.Println("Сервер запущен на http://localhost:6767")
	r.Run(":6767")
}

func SaveTrip(db *sql.DB, trip deftypes.Trip) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("ошибка транзакции: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.Exec(`INSERT OR IGNORE INTO users (name) VALUES (?)`, trip.User.Name)
	if err != nil {
		return fmt.Errorf("ошибка при сохранении пользователя: %w", err)
	}

	tripQuery := `INSERT INTO trips (date, place, user_name) VALUES (?, ?, ?)`
	result, err := tx.Exec(tripQuery, trip.Date, trip.Place, trip.User.Name)
	if err != nil {
		return fmt.Errorf("ошибка сохранения выезда: %w", err)
	}

	tripID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("неудача в получении ID поездки: %w", err)
	}

	for _, c := range trip.Catches {
		_, err := tx.Exec(`INSERT OR IGNORE INTO fishes (name) VALUES (?)`, c.Fish.Name)
		if err != nil {
			return fmt.Errorf("ошибка сохранения рыбы: %w", err)
		}
		_, err = tx.Exec(`INSERT OR IGNORE INTO lures (name) VALUES (?)`, c.Lure.Name)
		if err != nil {
			return fmt.Errorf("ошибка сохранения приманки: %w", err)
		}
		catchQuery := `SELECT catches (trip_id, fish_name, count, lure_name) VALUES (?, ?, ?, ?)`
		_, err = db.Exec(catchQuery)
		// Wait, keep the insert correct:
		catchQueryReal := `INSERT INTO catches (trip_id, fish_name, count, lure_name) VALUES (?, ?, ?, ?)`
		_, err = tx.Exec(catchQueryReal, tripID, c.Fish.Name, c.Count, c.Lure.Name)
		if err != nil {
			return fmt.Errorf("ошибка сохранения поимки рыбы: %w", err)
		}
	}

	return tx.Commit()
}

func DeleteTrip(db *sql.DB, tripID int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM catches WHERE trip_id = ?`, tripID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`DELETE FROM trips WHERE id = ?`, tripID)
	if err != nil {
		return err
	}

	return tx.Commit()
}
