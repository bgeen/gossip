# gossip
---

<img src="./assets/mascot.png" alt="Gossip mascot" width="200" height="200">

**gossip** is a lightweight, fast, and reliable Go module designed to interact with APIs from popular AI inference providers.

Its goal is to be ergonomic for developers while maintaining speed and reliability.

> ⚠️ Note: This module intentionally focuses on the most essential and commonly used features from each provider, rather than offering a full abstraction of every available API capability.

---

## ✅ Supported Providers & Features

### OpenAI
- ✅ Chat Completion
- ✅ Function Calling
- 🔜 Parallel Function Calling *(Coming Soon)*

### Anthropic
- ✅ Chat Completion
- ✅ Function Calling
- 🔜 Parallel Function Calling *(Coming Soon)*

### Groq
- 🔜 Support Coming Soon

---

# Usage

**Chat Completion**

```go
package main

import (
	"fmt"

	provider "go.bgeen.com/gossip"
)

func main() {
	agent, err := provider.NewAgent("openai:gpt-4o", provider.WithTemperature(0.8))
	if err != nil {
		fmt.Println(err)
		return
	}
	result := agent.Run("what is consciousness?")
	fmt.Println(result.Data)
}
```

<details>
<summary>Tool Calling</summary>

```go
package main

import (
	"fmt"
	"strings"

	provider "go.bgeen.com/gossip"
)

func main1() {
	agent, err := provider.NewAgent("anthropic:claude-3-5-sonnet-latest", provider.WithTemperature(0.8))
	if err != nil {
		fmt.Println(err)
		return
	}
	agent.RegisterTool(FindCityTemp, ParamsFindCityTemp{}, "find the weather temperature of the provided city name")
	result := agent.Run("whats the current temperature in delhi?")
	fmt.Println(result.Data)
}

type ParamsFindCityTemp struct {
	CityName string `json:"city_name" desctiption:"name of the city"`
}

func FindCityTemp(params ParamsFindCityTemp) string {
	if strings.ToLower(params.CityName) == "kolkata" {
		return "26 degree"
	}
	return "city not found"
}
```

</details>
