package command

import (
	"log"
	"os"

	"github.com/amatsagu/qord/api"
	"github.com/amatsagu/qord/rest"
)

var SendFile api.Command = api.Command{
	Name:        "send-file",
	Description: "Upload example image as message attachment.",
	SlashCommandHandler: func(itx *api.CommandInteraction) {
		imageFile, err := os.Open("./example-image.png")
		if err != nil {
			log.Println("failed to open image file:", err)
			return
		}
		defer imageFile.Close()

		files := []rest.File{
			{
				Name:   "example-image.png",
				Reader: imageFile,
			},
		}

		err = itx.SendReply(api.ResponseMessageData{
			Content: "This message should have attached files!",
		}, false, files)

		if err != nil {
			log.Println("SendReply failed:", err)
		}
	},
}
