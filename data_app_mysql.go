package main

import (
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"net/http"
)

func GetDataAppMysql(client *http.Client, data *Data) error {

	if !data.HasProcess("mysqld") {
		return nil
	}

	namespace := "Applications.MySQL."

	config := ReadConfig("MySQL")
	user := config.GetOrDefault("username", "")
	pass := config.GetOrDefault("password", "")

	if len(user) <= 0 || len(pass) <= 0 {
		fmt.Printf("Please provide MySQL username and password in /etc/linode/longview.d/MySQL.conf\n")
		return nil
	}

	mysqlConfig := mysql.Config{
		Addr:                 "localhost",
		User:                 user,
		Passwd:               pass,
		AllowNativePasswords: true,
	}
	mysqlDsn := mysqlConfig.FormatDSN()

	db, err := sql.Open("mysql", mysqlDsn)
	defer func() {
		if db != nil {
			_ = db.Close()
		}
	}()
	if err != nil {
		return nil
	}

	// Get data variables
	rows1, err := db.Query(`SHOW /*!50002 GLOBAL */ STATUS  WHERE Variable_name IN (
		"Com_select", "Com_insert", "Com_update", "Com_delete",
		"slow_queries",
		"Bytes_sent", "Bytes_received",
		"Connections", "Max_used_connections", "Aborted_Connects", "Aborted_Clients",
		"Qcache_queries_in_cache", "Qcache_hits", "Qcache_inserts", "Qcache_not_cached", "Qcache_lowmem_prunes")`)
	defer func() {
		if rows1 != nil {
			_ = rows1.Close()
		}
	}()

	if err == nil {
		for rows1.Next() {
			var key, value string
			err := rows1.Scan(&key, &value)
			if err == nil {
				if key == `Qcache_queries_in_cache` || key == `Max_used_connections` {
					data.Instant[namespace+key] = value
				} else {
					data.Longterm[namespace+key] = value
				}
			}
		}
	} else {
		fmt.Printf("Error executing query for rows1: %s\n", err)
	}

	// Get version variable
	var version string

	rows2, err := db.Query(`SHOW /*!50002 GLOBAL */ VARIABLES LIKE "version"`)
	defer func() {
		if rows2 != nil {
			_ = rows2.Close()
		}
	}()

	if err == nil {
		if rows2.Next() {
			var key, value string
			err := rows2.Scan(&key, &value)
			if err == nil {
				version = value
			}
		}
	} else {
		fmt.Printf("Error executing query for rows2: %s\n", err)
	}

	// Server version
	if len(version) > 0 {
		data.Instant[namespace+"version"] = version
	}

	// Overall status
	data.Instant[namespace+"status"] = 0
	data.Instant[namespace+"status_message"] = ""

	return nil
}
