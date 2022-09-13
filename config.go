package rmx

import "github.com/spf13/viper"

func LoadConfig() error {
	viper.SetConfigName("config") // name of config file (without extension)
	viper.SetConfigType("env")    // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath(".")      // optionally look for config in the working directory

	viper.SetDefault("PORT", "8080") // Set Default variables

	viper.AutomaticEnv()

	err := viper.ReadInConfig() // Find and read the config file
	if err != nil {             // Handle errors reading the config file
		return err
	}

	return nil
}
