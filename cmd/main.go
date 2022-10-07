package main

import (
	"log"
	"os"
	"time"

	"github.com/rog-golang-buddies/rmx/internal/commands"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

func main() {
	if err := initCLI().Run(os.Args); err != nil {
		log.Fatalln(err)
	}
}

func initCLI() *cli.App {
	c := &cli.App{
		EnableBashCompletion: true,
		Name:                 "rmx",
		Usage:                "RapidMidiEx Server CLI",
		Version:              "v0.0.1",
		Compiled:             time.Now().UTC(),
		Action: func(*cli.Context) error {
			return nil
		},
		Before: altsrc.InitInputSourceWithContext(
			commands.Flags,
			altsrc.NewYamlSourceFromFlagFunc("load"),
		),
		Commands: commands.Commands,
	}

	return c
}

// func getEnv(key, fallback string) string {
// 	if value, ok := os.LookupEnv(key); ok {
// 		return value
// 	}
// 	return fallback
// }

// func init() {
// 	// // name of config file (without extension)
// 	// viper.SetConfigName("config")
// 	// // REQUIRED if the config file does not have the extension in the name
// 	// viper.SetConfigType("env")
// 	// // optionally look for config in the working directory
// 	// viper.AddConfigPath(".")

// 	//// Set Default variables
// 	// viper.SetDefault("PORT", "8080")

// 	// viper.AutomaticEnv()

// 	// if err := viper.ReadInConfig(); err != nil {
// 	// 	panic(err)
// 	// }
// }

// // func LoadConfig(path string) (config Config, err error) {
// // 	// Read file path
// // 	viper.AddConfigPath(path)
// // 	// set config file and path
// // 	viper.SetConfigName("app")
// // 	viper.SetConfigType("env")
// // 	// watching changes in app.env
// // 	viper.AutomaticEnv()
// // 	// reading the config file
// // 	err = viper.ReadInConfig()
// // 	if err != nil {
// // 		return
// // 	}

// // 	err = viper.Unmarshal(&config)
// // 	return
// // }

// func loadConfig() error {
// 	_, b, _, _ := runtime.Caller(0)
// 	basepath := filepath.Join(filepath.Dir(b), "../")
// 	viper.SetConfigFile(basepath + ".env")
// 	// viper.AddConfigPath("../")
// 	viper.SetConfigType("dotenv")
// 	// viper.SetConfigFile(".env")

// 	return viper.ReadInConfig()
// }
