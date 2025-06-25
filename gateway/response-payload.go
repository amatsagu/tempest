package gateway

type IdentifyPayload struct {
	Opcode Opcode              `json:"op"`
	Data   IdentifyPayloadData `json:"d"`
}

type IdentifyPayloadData struct {
	Token          string                        `json:"token"`
	Intents        uint32                        `json:"intents"`
	ShardOrder     [2]uint16                     `json:"shard"`           // [currentID, maxCount]
	LargeThreshold uint8                         `json:"large_threshold"` // 50 - 250
	Properties     IdentifyPayloadDataProperties `json:"properties"`
}

type IdentifyPayloadDataProperties struct {
	OS      string `json:"os"`
	Browser string `json:"browser"`
	Device  string `json:"device"`
}
