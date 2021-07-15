	
	
// We must just proxy the request to the bctl-server
	// Make an Http client
	client := &http.Client{}

	// Build our request
	url := "https://bctl-server.bastionzero.com" + r.URL.Path
	req, _ := http.NewRequest(r.Method, url, nil)

	// Add the expected headers
	for name, values := range r.Header {
		// Loop over all values for the name.
		for _, value := range values {
			req.Header.Set(name, value)
		}
	}

	// Add our custom headers
	req.Header.Set("X-KUBE-ROLE-IMPERSONATE", "cwc-dev-developer")
	req.Header.Set("Authorization", "Bearer 1234")

	// Set any query params
	for key, values := range r.URL.Query() {
		for value := range values {
			req.URL.Query().Add(key, values[value])
		}
	}

	// Make the request and wait for the body to close
	res, _ := client.Do(req)
	defer res.Body.Close()

	if res.StatusCode == http.StatusOK {
		// Loop over all headers, and add them to our response back to kubectl
		for name, values := range res.Header {
			for _, value := range values {
				w.Header().Set(name, value)
			}
		}

		// Get all the body
		bodyBytes, err := ioutil.ReadAll(res.Body)
		if err != nil {
			fmt.Println("FATAL ERROR!")
		}
		// Write the body to the response to kubectl
		w.Write(bodyBytes)

	} else {
		fmt.Printf("Invalid status code returned from request: %d\n", res.StatusCode)
	}