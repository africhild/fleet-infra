package storage

import (
    "bufio"
    "fmt"
    "os"
    "strconv"
	"strings"
)

const portFilePath = "ports.txt"
const minPort = 8000
const maxPort = 9000

// Check if a port is in the file
func IsPortUsed(port int) bool {
    file, err := os.Open(portFilePath)
    if err != nil {
        return false
    }
    defer file.Close()
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
		usedPort, _ := strconv.Atoi(strings.TrimSpace(strings.Split(scanner.Text(), ":")[1]))
        if usedPort == port {
            return true
        }
    }
    return false
}

// Add a port to the file
func AddPort(appName string, port int) error {
    if IsPortUsed(port) {
        // return fmt.Errorf("port %d is already used", port)
		return nil
    }

    file, err := os.OpenFile(portFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer file.Close()

    _, err = file.WriteString(fmt.Sprintf("%s: %d\n", appName, port))
    return err
}

// Remove a port from the file
func RemovePort(appName string, port int) error {
	file, err := os.ReadFile(portFilePath)
    if err != nil {
        return err
    }

    lines := strings.Split(string(file), "\n")
    var newLines []string
    for _, line := range lines {
        if line != fmt.Sprintf("%s: %d\n", appName, port) || line  != fmt.Sprintf("%s:%d\n", appName, port){
            newLines = append(newLines, line)
        }
    }

    return os.WriteFile(portFilePath, []byte(strings.Join(newLines, "\n")), 0644)
}


// Allocate a new port
func AllocatePort(appName string) (int, error) {
    for port := minPort; port <= maxPort; port++ {
        if !IsPortUsed(port) {
            err := AddPort(appName, port)
            if err != nil {
                return 0, err
            }
            return port, nil
        }
    }
    return 0, fmt.Errorf("no available ports in the range")
}