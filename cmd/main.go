package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/zhanserikAmangeldi/block/dto"
)

var db *sql.DB

func main() {
	var err error
	db, err = sql.Open("sqlite3", "./test.db")

	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	if err = db.Ping(); err != nil {
		log.Fatal("Failed to ping database:", err)
	}

	if err = createTables(); err != nil {
		log.Fatal("Failed to create tables:", err)
	}

	// for i := 0; i < 100; i++ {
	// 	blockData := getNowBlock()
	// 	if err = saveBlockData(blockData); err != nil {
	// 		log.Fatal("Failed to save block data:", err)
	// 	}
	// }

	startBlockMonitoring(time.Second * 10)

	fmt.Println("Block data saved successfully!")
}

func startBlockMonitoring(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			blockData := getNowBlock()
			if err := saveBlockData(blockData); err != nil {
				log.Printf("Error saving block data: %v", err)
			}
		}
	}
}

func getNowBlock() []byte {
	url := "https://api.shasta.trongrid.io/wallet/getnowblock"

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		log.Fatal("Failed to create request:", err)
	}

	req.Header.Add("accept", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Fatal("Failed to make request:", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Fatal("Failed to read response:", err)
	}

	return body
}

func createTables() error {
	blocksTable := `
	CREATE TABLE IF NOT EXISTS blocks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		block_id TEXT UNIQUE NOT NULL,
		block_number INTEGER NOT NULL,
		tx_trie_root TEXT,
		witness_address TEXT,
		parent_hash TEXT,
		version INTEGER,
		timestamp INTEGER,
		witness_signature TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	transactionsTable := `
	CREATE TABLE IF NOT EXISTS transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tx_id TEXT UNIQUE NOT NULL,
		block_id TEXT NOT NULL,
		contract_ret TEXT,
		signature TEXT,
		ref_block_bytes TEXT,
		ref_block_hash TEXT,
		expiration INTEGER,
		timestamp INTEGER,
		raw_data_hex TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (block_id) REFERENCES blocks(block_id)
	);`

	contractsTable := `
	CREATE TABLE IF NOT EXISTS contracts (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		tx_id TEXT NOT NULL,
		contract_type TEXT,
		type_url TEXT,
		contract_value TEXT, -- JSON stored as TEXT in SQLite
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (tx_id) REFERENCES transactions(tx_id)
	);`

	if _, err := db.Exec(blocksTable); err != nil {
		return fmt.Errorf("failed to create blocks table: %v", err)
	}

	if _, err := db.Exec(transactionsTable); err != nil {
		return fmt.Errorf("failed to create transactions table: %v", err)
	}

	if _, err := db.Exec(contractsTable); err != nil {
		return fmt.Errorf("failed to create contracts table: %v", err)
	}

	return nil
}

func saveBlockData(jsonData []byte) error {
	var blockResp dto.BlockResponse
	if err := json.Unmarshal(jsonData, &blockResp); err != nil {
		return fmt.Errorf("failed to parse JSON: %v", err)
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	blockQuery := `
		INSERT OR IGNORE INTO blocks (block_id, block_number, tx_trie_root, witness_address, parent_hash, version, timestamp, witness_signature)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = tx.Exec(blockQuery,
		blockResp.BlockID,
		blockResp.BlockHeader.RawData.Number,
		blockResp.BlockHeader.RawData.TxTrieRoot,
		blockResp.BlockHeader.RawData.WitnessAddress,
		blockResp.BlockHeader.RawData.ParentHash,
		blockResp.BlockHeader.RawData.Version,
		blockResp.BlockHeader.RawData.Timestamp,
		blockResp.BlockHeader.WitnessSignature,
	)
	if err != nil {
		return fmt.Errorf("failed to insert block: %v", err)
	}

	for _, transaction := range blockResp.Transactions {
		contractRet := ""
		if len(transaction.Ret) > 0 {
			contractRet = transaction.Ret[0].ContractRet
		}

		signature := ""
		if len(transaction.Signature) > 0 {
			signature = transaction.Signature[0]
		}

		txQuery := `
			INSERT OR IGNORE INTO transactions (tx_id, block_id, contract_ret, signature, ref_block_bytes, ref_block_hash, expiration, timestamp, raw_data_hex)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

		_, err = tx.Exec(txQuery,
			transaction.TxID,
			blockResp.BlockID,
			contractRet,
			signature,
			transaction.RawData.RefBlockBytes,
			transaction.RawData.RefBlockHash,
			transaction.RawData.Expiration,
			transaction.RawData.Timestamp,
			transaction.RawDataHex,
		)

		if err != nil {
			return fmt.Errorf("failed to insert transaction %s: %v", transaction.TxID, err)
		}

		for _, contract := range transaction.RawData.Contract {
			contractValueJSON, err := json.Marshal(contract.Parameter.Value)
			if err != nil {
				log.Printf("Warning: failed to marshal contract value for tx %s: %v", transaction.TxID, err)
				continue
			}

			contractQuery := `
				INSERT INTO contracts (tx_id, contract_type, type_url, contract_value)
				VALUES (?, ?, ?, ?)`

			_, err = tx.Exec(contractQuery,
				transaction.TxID,
				contract.Type,
				contract.Parameter.TypeURL,
				string(contractValueJSON),
			)
			if err != nil {
				return fmt.Errorf("failed to insert contract for tx %s: %v", transaction.TxID, err)
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	fmt.Printf("Saved block %d with %d transactions\n",
		blockResp.BlockHeader.RawData.Number,
		len(blockResp.Transactions))

	return nil
}
