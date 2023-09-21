package tempest

import (
	"testing"

	"github.com/sugawarayuuta/sonnet"
)

type exyyy struct {
	Message string `json:"message"`
}

type exxxx struct {
	Test map[Snowflake]exyyy `json:"test"`
}

func TestSonnet(t *testing.T) {
	// Try simple
	var data1 exxxx
	body1 := []byte(`{
		"test": {
		  "1010": {"message": "hello"}
		}
	  }`)

	if err := sonnet.Unmarshal(body1, &data1); err != nil {
		t.Error(err)
	}

	// Try advanced
	var data2 CommandInteractionData
	body2 := []byte(`{
		"guild_id": "76543210987654321",
		"id": "1151765326615289907",
		"name": "hello",
		"options": [
		  {
			"name": "message",
			"type": 3,
			"value": "<@12345678901234567> is great"
		  }
		],
		"resolved": {
		  "members": {
			"12345678901234567": {
			  "avatar": null,
			  "communication_disabled_until": null,
			  "flags": 0,
			  "joined_at": "2022-01-01T06:21:57.248000+00:00",
			  "nick": "Drew",
			  "pending": false,
			  "permissions": "562949953421311",
			  "premium_since": null,
			  "roles": [
				"12345678901234568",
				"12345678901234569"
			  ],
			  "unusual_dm_activity_until": null
			}
		  }
		},
		"users": {}
	  }`)

	if err := sonnet.Unmarshal(body2, &data2); err != nil {
		t.Error(err)
	}
}
