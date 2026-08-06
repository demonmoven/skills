# 🔄 Add Init Logic for RAL Framework

## 📋 Overview

> **IMPORTANT**: Resource access codes in this project have been refactored to use RAL Framework. You now need to add proper initialization logic.

This guide will help you implement the RAL (Resource Access Layer) framework initialization logic in your project. The implementation process varies depending on your project type.

## 🔍 Step 1: Identify Your Project Type

### GDP Project
Your project uses GDP framework if it has imports like:
```go
import "code.byted.org/gdp/gdp"
```

### Non-GDP Project
Your project is non-GDP if it uses any of these frameworks:
- Hertz
- Ginex
- Kitex
- Kitx
- FaaS
- Others

## 🛠️ Step 2: Implementation Instructions

### For GDP Projects

#### Implementation Checklist
- [ ] Identify initialization code in main.go
- [ ] Move initialization logic (excluding plugin registration) to WithBootstrapFn parameter

#### Example Implementation

**Before:**
```go
package main

import (
    // GDP imports
    "code.byted.org/gdp/gdp"
    // Other imports
    "code.byted.org/tiktok/yourproject/pkg/app"
)

func main() {
    app.Init()
    
    // GDP initialization code
    // ...
}
```

**After:** (Choose one option)

```go
package main

import (
    // GDP imports
    "code.byted.org/gdp/gdp"
    "code.byted.org/gdp/regionrouter"
    // Other imports
    "code.byted.org/tiktok/yourproject/pkg/app"
)

func main() {
    regionrouter.UseRalPlugin() // Put it ahead.
	
    gdp.Init(
        // Other GDP options
        gdp.WithBootstrapFn(app.Init), // Move all initialization logic here
    ).Run()
}
```

### For Non-GDP Projects

> **CRITICAL NOTES**:
> - Always add RAL initialization at the **beginning** of the main function， before any other Init functions.
> - The func main entry point typically resides in main.go, but not exclusively. Locate the entry point in the current directory.

#### Framework-Specific Implementation

<details>
<summary><b>Hertz Framework</b></summary>

**Implementation Checklist**
- [ ] Add bridge and RAL imports
- [ ] Initialize bridge with RAL support
- [ ] Set up Hertz-specific middleware
- [ ] Update build.sh

**Code Changes:**
```go
// Before
package main

import (
    "code.byted.org/middleware/hertz/byted"
)

func main() {
    // Other Logics
    
    r := byted.Default()
    // Register routes
    register(r)
    r.Spin()
}
```

```go
// After
package main

import (
	"code.byted.org/gdp/dam/bridge"                   // New import
	bridgeHertz "code.byted.org/gdp/dam/bridge/hertz" // New import
	"code.byted.org/middleware/hertz/byted"
)

func main() {
    bridge.Init(bridge.WithRal(), bridge.WithRegionrouter()) // Add RAL initialization

	byted.Init()
	r := byted.Default()

	mw := bridgeHertz.NewMiddleware()
	r.Use(mw)

	// Other logic follows after RAL initialization
	
	register(r)
	r.Spin()
}
```

Or

```go
// Before
package main

import (
	"code.byted.org/middleware/hertz"
    "code.byted.org/middleware/hertz/app" // the difference with above is that this import is app package, not byted package
)

func main() {
    // Other Logics
	
	hertz.Init()
    r := app.Default()
    // Register routes
    register(r)
    r.Spin()
}
```

```go
// After
package main

import (
	"code.byted.org/gdp/dam/bridge"                     // New import
	bridgeHertz "code.byted.org/gdp/dam/bridge/hertzv0" // New import. Corresponding to "code.byted.org/middleware/hertz/app"
   "code.byted.org/middleware/hertz"
	"code.byted.org/middleware/hertz/app"
)

func main() {
    bridge.Init(bridge.WithRal(), bridge.WithRegionrouter()) // Add RAL initialization

    hertz.Init()                                                                                                                                                                                                                                                                               │
    r := app.Default()
	
	mw := bridgeHertz.NewMiddleware()
	r.Use(mw)

	// Other logic follows after RAL initialization
	
	register(r)
	r.Spin()
}
```
</details>

<details>
<summary><b>Ginex Framework</b></summary>

**Implementation Checklist**
- [ ] Add bridge and RAL imports
- [ ] Initialize bridge with RAL support
- [ ] Set up Ginex-specific bridge
- [ ] Update build.sh

**Code Changes:**
```go
// Before
package main

import (
    "os"
    "code.byted.org/gin/ginex"
)

func main() {
    ginex.Init()
    r := ginex.Default()
    
    register(r)

    if err := r.Run(); err != nil {
       os.Exit(1)
    }
}
```

```go
// After
package main

import (
    "os"
    "code.byted.org/gin/ginex"
    "code.byted.org/gdp/dam/bridge"  // New import
    bridgeGinex "code.byted.org/gdp/dam/bridge/ginex"  // New import
)

func main() {
    bridge.Init(bridge.WithRal(), bridge.WithRegionrouter()) // Add RAL initialization
	
    ginex.Init()
    r := ginex.Default()
	
    bridgeGinex.Use(r)
    
    register(r)
    
    if err := r.Run(); err != nil {
       os.Exit(1)
    }
}
```
</details>

<details>
<summary><b>Kitex Framework</b></summary>

**Implementation Checklist**
- [ ] Add bridge and RAL imports
- [ ] Initialize bridge with RAL support
- [ ] Set up Kitex-specific middleware
- [ ] Update build.sh

**Code Changes:**
```go
// Before
package main

import (
    x "code.byted.org/x/x/kitex_gen/x/x"
)

func main() {
    svr := x.NewServer(new(XImpl), server.WithMiddleware(mwX))
    err := svr.Run()
    if err != nil {
        panic(err)
    }
}
```

```go
// After
package main

import (
    x "code.byted.org/x/x/kitex_gen/x/x"
    "code.byted.org/gdp/dam/bridge"  // New import
    bridgeKitex "code.byted.org/gdp/dam/bridge/kitex"  // New import
    "code.byted.org/kite/kitex/server"  // New import
)

func main() {
    bridge.Init(bridge.WithRal(), bridge.WithRegionrouter()) // Add RAL initialization
    
    mw := bridgeKitex.NewMiddleware()
    svr := x.NewServer(new(XImpl), server.WithMiddleware(mw), server.WithMiddleware(mwX)) // server.WithMiddleware only accept one middleware
    
    err := svr.Run()
    if err != nil {
        panic(err)
    }
}
```
</details>

<details>
<summary><b>Kite Framework</b></summary>

**Implementation Checklist**
- [ ] Add bridge and RAL imports
- [ ] Initialize bridge with RAL support
- [ ] Set up Kite-specific middleware
- [ ] Update build.sh

**Code Changes:**
```go
// Before
package main

import (
    "code.byted.org/gopkg/logs"
    "code.byted.org/kite/kite"
)

func main() {
    kite.Init()
    
    logs.Error("%v", kite.Run())
    logs.Stop()
}
```

```go
// After
package main

import (
    "code.byted.org/gopkg/logs"
    "code.byted.org/kite/kite"
    "code.byted.org/gdp/dam/bridge"  // New import
    bridgeKite "code.byted.org/gdp/dam/bridge/kite"  // New import
)

func main() {
    bridge.Init(bridge.WithRal(), bridge.WithRegionrouter()) // Add RAL initialization
	
    kite.Use(bridgeKite.NewMiddleware())

    kite.Init()
    
    logs.Error("%v", kite.Run())
    logs.Stop()
}
```
</details>

<details>
<summary><b>FaaS Framework</b></summary>

**Implementation Checklist**
- [ ] Add bridge and RAL imports
- [ ] Initialize bridge with RAL support
- [ ] Wrap handler with FaaS adapter
- [ ] Update build.sh

**Code Changes:**
```go
// Before
package main

import (
    "context"
    "code.byted.org/bytefaas/faas-go/bytefaas"
    "code.byted.org/bytefaas/faas-go/events"
)

func handler(ctx context.Context, r *events.HTTPRequest) (*events.EventResponse, error) {
   // Your handler logic
}

func main() {
    bytefaas.Start(handler)
}
```

```go
// After
package main

import (
    "context"
    "code.byted.org/bytefaas/faas-go/bytefaas"
    "code.byted.org/bytefaas/faas-go/events"
    "code.byted.org/gdp/dam/bridge"  // New import
    "code.byted.org/gdp/dam/bridge/faas"  // New import
)

func handler(ctx context.Context, r *events.HTTPRequest) (*events.EventResponse, error) {
   // Your handler logic
}

func main() {
    bridge.Init(bridge.WithRal(), bridge.WithRegionrouter()) // Add RAL initialization
	
    bytefaas.Start(faas.Wrap(handler))
}
```
</details>

<details>
<summary><b>Consumer</b></summary>

**Implementation Checklist**
- [ ] Add bridge and RAL imports
- [ ] Initialize bridge with RAL support
- [ ] Add InitContext at the beginning of message handler function
- [ ] Update build.sh

**Code Changes:**
```go
// Before
package main

func main() {
	// ...

    // Example consumer code
    c, err := eventbus.NewConsumer(conf, handler)
    if err != nil {
        panic(err)
    }

    if err = c.Run(); err != nil {
        panic(err)
    }
}

func handler(ctx context.Context, event *eventbus.ConsumerEvent) error { 
	// ...
}
```

```go
// After
package main

import (
   eventbus "code.byted.org/eventbus/client-go"
    "code.byted.org/gdp/dam/bridge"  // New import
)

func main() {
    bridge.Init(bridge.WithRal(), bridge.WithRegionrouter()) // Add RAL initialization
    
	// ...
}

func handler(ctx context.Context, event *eventbus.ConsumerEvent) error {
	ctx = bridge.InitContext(ctx) // Add InitContext at the beginning of message handler function
	
	// ...
}
```
</details>


<details>
<summary><b>Others</b></summary>

**Implementation Checklist**
- [ ] Add bridge and RAL imports
- [ ] Initialize bridge with RAL support
- [ ] Update build.sh 

**Code Changes:**
```go
// Before
package main

func main() {
    // ...
}
```

```go
// After
package main

import (
    "code.byted.org/gdp/dam/bridge"  // New import
)

func main() {
    bridge.Init(bridge.WithRal(), bridge.WithRegionrouter()) // Add RAL initialization
	
    // ....
}
```
</details>

## 📦 Step 3: Update build.sh (If build.sh is given)

<details>
<summary><b>For GDP Projects</b></summary>

Please verify and update the build.sh file to properly handle RAL configuration files with these requirements:

1. build.sh must copy ALL files from `conf/ral` to `output/conf/ral`
2. The copy operation should:
    - Preserve the directory structure (`conf/ral` → `output/conf/ral`)
    - Overwrite existing files (`-f` flag)
    - Recursively copy directories (`-r` flag)
3. The implementation should:
    - First create the necessary output directory structure
    - Handle cases where `conf/ral` might not exist
4. The solution doesn't need to match the example exactly, but must meet all functional requirements
5. Don't change unrelated parts in build.sh

Example implementation (for reference):
```sh
# Existing code...

conf='conf/app'

mkdir -p output/${conf}

# Existing code...

ls conf | grep -v ral | xargs -I {} cp -r conf/{} output/${conf} 2>/dev/null
if [ -d "conf/ral" ]; then
    cp -fr conf/ral output/conf
fi

# Remaining code...
```
</details>

<details>
<summary><b>For Hertz and Kitex Projects</b></summary>

Update build.sh to copy the RAL configuration:

```sh
# Existing code...

# Copy configuration files with recursive flag
cp -r conf/* output/conf/

# Remaining code...
```
</details>

<details>
<summary><b>For Ginex and Kitx Projects</b></summary>

Update build.sh to specifically handle RAL configuration:

```sh
# Existing code...

find conf/ -type f ! -name "*_local.*" | xargs -I{} cp {} output/conf/

# Additionally, ensure RAL config files are copied
mkdir -p output/conf/ral
cp -r conf/ral/* output/conf/ral

# Remaining code...
```

or

```sh
# Existing code...

# If original code is like `cp conf/* ${OUT_PUT_DIR}/conf/`, change it to
cp -r conf/* ${OUT_PUT_DIR}/conf/

# Remaining code...
```
</details>

<details>
<summary><b>For Others</b></summary>

Update build.sh flexibly to ensure that files under conf/ral are copied to output/conf/ral.

Example:
```sh
# Existing code...

# Maintain original conf copy logic
# Besides, create RAL configuration directory and copy conf/ral files
mkdir -p output/conf/ral
cp -r conf/ral/* output/conf/ral

# Remaining code...
```
</details>

## ✅ Final Verification

Ensure you've completed all these steps:
1. Identified your project type correctly (GDP vs Non-GDP)
2. Added all required imports for your framework
3. Placed RAL initialization at the beginning of your main function
4. Added framework-specific middleware configuration
5. Updated build.sh to handle RAL configuration files

Extra requirements:
**Preservation Rules**: Keep unrelated comments and code structures intact.