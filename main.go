// Item Instance conversion for cMaNGOS
// Author: Henhouse
// Based on C++ project: https://github.com/vmangos/ItemInstance
package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"io/ioutil"
	"log"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("\n\n####################################################")
	fmt.Println("# BACKUP YOUR DATABASE BEFORE RUNNING THIS SCRIPT! #")
	fmt.Println("####################################################")
	fmt.Println()

	// Prompt user for DB info and connect.
	db := HandleConnectDB()

	// Count entries to convert so we know what size to make the array.
	var totalEntries uint
	row := db.QueryRow("SELECT COUNT(*) FROM `item_instance`")
	if err := row.Scan(&totalEntries); err != nil {
		panic(err)
	}

	if totalEntries == 0 {
		fmt.Println("No entries in `item_instance`. Exiting.")
		return
	}

	fmt.Println("Loading all item instance entries...")
	rows, err := db.Query("SELECT `data` FROM `item_instance`")
	if err != nil {
		panic(err)
	}

	// Create slice with capacity for all entries.
	blobResults := make([]string, 0, totalEntries)

	var data string
	for rows.Next() {
		err = rows.Scan(&data)
		if err != nil {
			panic(err)
		}

		blobResults = append(blobResults, data)
	}
	fmt.Printf("Collected %v entries to be converted.\n", len(blobResults))

	fmt.Println("Beginning parse...")

	// Way faster than concatenating strings
	var entireQuery strings.Builder

	entireQuery.WriteString("TRUNCATE `item_instance`;\n")

	entireQuery.WriteString("ALTER TABLE `item_instance` DROP `data`;\n\n")

	entireQuery.WriteString("ALTER TABLE `item_instance`\n")
	entireQuery.WriteString(" ADD `itemEntry` MEDIUMINT(8) UNSIGNED NOT NULL DEFAULT '0' AFTER `owner_guid`,\n")
	entireQuery.WriteString(" ADD `creatorGuid` INT(10) UNSIGNED NOT NULL DEFAULT '0' AFTER `itemEntry`,\n")
	entireQuery.WriteString(" ADD `giftCreatorGuid` INT(10) UNSIGNED NOT NULL DEFAULT '0' AFTER `creatorGuid`,\n")
	entireQuery.WriteString(" ADD `count` INT(10) UNSIGNED NOT NULL DEFAULT '1' AFTER `giftCreatorGuid`,\n")
	entireQuery.WriteString(" ADD `duration` INT(10) UNSIGNED NOT NULL AFTER `count`,\n")
	entireQuery.WriteString(" ADD `charges` TEXT NOT NULL AFTER `duration`,\n")
	entireQuery.WriteString(" ADD `flags` INT(10) UNSIGNED NOT NULL DEFAULT '0' AFTER `charges`,\n")
	entireQuery.WriteString(" ADD `enchantments` TEXT NOT NULL AFTER `flags`,\n")
	entireQuery.WriteString(" ADD `randomPropertyId` INT(11) NOT NULL DEFAULT '0' AFTER `enchantments`,\n")
	entireQuery.WriteString(" ADD `durability` INT(10) UNSIGNED NOT NULL DEFAULT '0' AFTER `randomPropertyId`,\n")
	entireQuery.WriteString(" ADD `itemTextId` MEDIUMINT(8) UNSIGNED NOT NULL DEFAULT '0' AFTER `durability`;\n\n")

	entireQuery.WriteString("INSERT INTO `item_instance` VALUES \n")

	for i := range blobResults {
		blob := blobResults[i]

		entireQuery.WriteString(ParseDataBlob(blob))

		if i != len(blobResults)-1 {
			entireQuery.WriteString(",")
		} else {
			entireQuery.WriteString(";")
		}
		entireQuery.WriteString("\n")
	}
	fmt.Println("Done parsing.")
	db.Close()

	fmt.Println("Writing full query to file: item_instance_converted.sql")

	queryAsBytes := bytes.NewBufferString(entireQuery.String()).Bytes()
	ioutil.WriteFile("item_instance_converted.sql", queryAsBytes, 0644)

	fmt.Println("Done.")
}

func ParseDataBlob(blob string) string {
	values := strings.Split(blob, " ")

	var itemGuid = values[0]
	var itemEntry = values[3]
	var ownerGuid = uint64(stringToUint32(values[6]))
	var creatorGuid = uint64(stringToUint32(values[10]))
	var giftCreator = uint64(stringToUint32(values[12]))
	var stackCount = values[14]
	var duration = values[15]

	var spellCharges string
	for i := 16; i < 21; i++ {
		if i != 16 {
			spellCharges += " "
		}

		// Stored as uint32 but we need to cast to int32.
		spellCharges += fmt.Sprintf("%d", int32(stringToUint32(values[i])))
	}

	var flags = values[21]

	var enchantments string
	for i := 22; i < 55; i++ {
		if i != 22 {
			enchantments += " "
		}

		// Use stringValues here since they're already as strings
		enchantments += values[i]
	}

	var randomPropertyId = int32(stringToUint32(values[56])) // Stored as uint32, but we want int32 representation.
	var textId = values[57]
	var durability = values[58]

	return " (" +
		itemGuid + ", " +
		fmt.Sprintf("%d", ownerGuid) + ", " +
		itemEntry + ", " +
		fmt.Sprintf("%d", creatorGuid) + ", " +
		fmt.Sprintf("%d", giftCreator) + ", " +
		stackCount + ", " +
		duration + ", " +
		"'" + spellCharges + "'" + ", " +
		flags + ", " +
		"'" + enchantments + "'" + ", " +
		fmt.Sprintf("%d", randomPropertyId) + ", " +
		durability + ", " +
		textId +
		")"
}

func stringToUint32(str string) uint32 {
	result, err := strconv.ParseUint(str, 10, 32)
	if err != nil {
		panic(err)
	}

	return uint32(result)
}

func HandleConnectDB() *sql.DB {
	var host, port, user, pass, database string

	fmt.Println("Host:")
	fmt.Scanf("%s", &host)

	fmt.Println("Port:")
	fmt.Scanf("%s", &port)

	fmt.Println("User:")
	fmt.Scanf("%s", &user)

	fmt.Println("Password:")
	fmt.Scanf("%s", &pass)

	fmt.Println("Database:")
	fmt.Scanf("%s", &database)

	var connStr string
	connStr += user
	connStr += ":" + pass
	connStr += "@tcp(" + host + ":" + port + ")"
	connStr += "/" + database

	session, err := sql.Open("mysql", connStr)
	if err != nil {
		log.Fatal("Couldn't connect to DB: ", err)
	}

	// sql.Open above does not return connection failure, so we must ping to ensure the session is valid.
	err = session.Ping()
	if err != nil {
		log.Fatal("Couldn't connect to DB:", err)
	}

	return session
}
