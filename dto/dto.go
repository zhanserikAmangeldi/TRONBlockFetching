package dto

type BlockResponse struct {
	BlockID      string        `json:"blockID"`
	BlockHeader  BlockHeader   `json:"block_header"`
	Transactions []Transaction `json:"transactions"`
}

type BlockHeader struct {
	RawData          RawBlockData `json:"raw_data"`
	WitnessSignature string       `json:"witness_signature"`
}

type RawBlockData struct {
	Number         int64  `json:"number"`
	TxTrieRoot     string `json:"txTrieRoot"`
	WitnessAddress string `json:"witness_address"`
	ParentHash     string `json:"parentHash"`
	Version        int    `json:"version"`
	Timestamp      int64  `json:"timestamp"`
}

type Transaction struct {
	Ret        []TransactionResult `json:"ret"`
	Signature  []string            `json:"signature"`
	TxID       string              `json:"txID"`
	RawData    RawTransactionData  `json:"raw_data"`
	RawDataHex string              `json:"raw_data_hex"`
}

type TransactionResult struct {
	ContractRet string `json:"contractRet"`
}

type RawTransactionData struct {
	Contract      []Contract `json:"contract"`
	RefBlockBytes string     `json:"ref_block_bytes"`
	RefBlockHash  string     `json:"ref_block_hash"`
	Expiration    int64      `json:"expiration"`
	Timestamp     int64      `json:"timestamp"`
}

type Contract struct {
	Parameter ContractParameter `json:"parameter"`
	Type      string            `json:"type"`
}

type ContractParameter struct {
	Value   map[string]interface{} `json:"value"`
	TypeURL string                 `json:"type_url"`
}
