package main

import (
	"fmt"

	"github.com/afroximity/nsfw_detection"

	"github.com/sirupsen/logrus"
)

func main() {
	logrus.SetLevel(logrus.InfoLevel)

	predictor, err := nsfw.NewLatestPredictor()
	if err != nil {
		logrus.Fatal("unable to create predictor", err)
	}

	image := predictor.NewImage("./example/dog.jpg", 3)

	fmt.Println(predictor.Predict(image).Describe())
}
