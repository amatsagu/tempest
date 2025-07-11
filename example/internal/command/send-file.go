package command

import (
	"log"
	"os"
	"time"

	tempest "github.com/amatsagu/tempest"
)

var SendFile tempest.Command = tempest.Command{
	Name:        "send-file",
	Description: "Upload example image as message attachment.",
	SlashCommandHandler: func(itx *tempest.CommandInteraction) {
		start := time.Now()

		imageFile, err := os.Open("./example-image.png")
		if err != nil {
			log.Println("failed to open image file:", err)
			return
		}
		defer imageFile.Close()

		log.Println("Opening files took:", time.Since(start))

		files := []tempest.File{
			{
				Name:   "example-image.png",
				Reader: imageFile,
			},
		}

		err = itx.SendReply(tempest.ResponseMessageData{
			Content: "This message should have attached files!",
		}, false, files)
		if err != nil {
			log.Println("SendReply failed:", err)
		}

		log.Println("SendReply took:", time.Since(start))
	},
}
