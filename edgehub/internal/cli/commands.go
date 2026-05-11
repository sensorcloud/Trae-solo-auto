package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/spf13/cobra"
)

var (
	serverAddr string
	apiKey     string
	outputJSON bool
)

func NewRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edge",
		Short: "EdgeHub CLI",
		Long:  "Command-line interface for EdgeHub platform",
	}

	cmd.PersistentFlags().StringVar(&serverAddr, "server", "http://localhost:8080", "API server address")
	cmd.PersistentFlags().StringVar(&apiKey, "api-key", "", "API key for authentication")
	cmd.PersistentFlags().BoolVarP(&outputJSON, "json", "j", false, "Output in JSON format")

	return cmd
}

func NewNodeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Manage edge nodes",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all nodes",
		RunE:  runNodeList,
	}
	listCmd.Flags().String("status", "", "Filter by status")
	listCmd.Flags().String("region", "", "Filter by region")
	listCmd.Flags().Bool("has-gpu", false, "Show only GPU nodes")

	registerCmd := &cobra.Command{
		Use:   "register",
		Short: "Register a new node",
		RunE:  runNodeRegister,
	}
	registerCmd.Flags().String("name", "", "Node name")
	registerCmd.Flags().String("cluster", "", "Cluster ID")

	getCmd := &cobra.Command{
		Use:   "get [node-id]",
		Short: "Get node details",
		Args:  cobra.ExactArgs(1),
		RunE:  runNodeGet,
	}

	deleteCmd := &cobra.Command{
		Use:   "delete [node-id]",
		Short: "Delete a node",
		Args:  cobra.ExactArgs(1),
		RunE:  runNodeDelete,
	}

	cmd.AddCommand(listCmd, registerCmd, getCmd, deleteCmd)
	return cmd
}

func NewJobCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "job",
		Short: "Manage compute jobs",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all jobs",
		RunE:  runJobList,
	}
	listCmd.Flags().String("status", "", "Filter by status")
	listCmd.Flags().String("type", "", "Filter by type")
	listCmd.Flags().String("queue", "", "Filter by queue")

	submitCmd := &cobra.Command{
		Use:   "submit",
		Short: "Submit a new job",
		RunE:  runJobSubmit,
	}
	submitCmd.Flags().String("file", "job.yaml", "Job definition file")
	submitCmd.Flags().String("name", "", "Job name")
	submitCmd.Flags().String("image", "", "Container image")
	submitCmd.Flags().StringArray("cmd", []string{}, "Command to run")
	submitCmd.Flags().String("cpu", "1", "CPU request")
	submitCmd.Flags().String("memory", "1Gi", "Memory request")
	submitCmd.Flags().String("gpu", "0", "GPU request")

	getCmd := &cobra.Command{
		Use:   "get [job-id]",
		Short: "Get job details",
		Args:  cobra.ExactArgs(1),
		RunE:  runJobGet,
	}

	logsCmd := &cobra.Command{
		Use:   "logs [job-id]",
		Short: "Get job logs",
		Args:  cobra.ExactArgs(1),
		RunE:  runJobLogs,
	}

	stopCmd := &cobra.Command{
		Use:   "stop [job-id]",
		Short: "Stop a running job",
		Args:  cobra.ExactArgs(1),
		RunE:  runJobStop,
	}

	deleteCmd := &cobra.Command{
		Use:   "delete [job-id]",
		Short: "Delete a job",
		Args:  cobra.ExactArgs(1),
		RunE:  runJobDelete,
	}

	cmd.AddCommand(listCmd, submitCmd, getCmd, logsCmd, stopCmd, deleteCmd)
	return cmd
}

func NewMarketCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "market",
		Short: "Manage computing power market",
	}

	listOffersCmd := &cobra.Command{
		Use:   "list-offers",
		Short: "List available offers",
		RunE:  runMarketListOffers,
	}
	listOffersCmd.Flags().String("region", "", "Filter by region")
	listOffersCmd.Flags().Float64("min-cpu", 0, "Minimum CPU cores")
	listOffersCmd.Flags().Int("min-gpu", 0, "Minimum GPU count")
	listOffersCmd.Flags().Float64("max-price", 0, "Maximum price per hour")

	createOfferCmd := &cobra.Command{
		Use:   "create-offer",
		Short: "Create a new offer",
		RunE:  runMarketCreateOffer,
	}

	listPricesCmd := &cobra.Command{
		Use:   "list-prices",
		Short: "List current prices",
		RunE:  runMarketListPrices,
	}

	recommendCmd := &cobra.Command{
		Use:   "recommend",
		Short: "Get price recommendations",
		RunE:  runMarketRecommend,
	}
	recommendCmd.Flags().String("region", "", "Preferred region")
	recommendCmd.Flags().Float64("min-cpu", 0, "Minimum CPU cores")
	recommendCmd.Flags().Int("min-gpu", 0, "Minimum GPU count")

	cmd.AddCommand(listOffersCmd, createOfferCmd, listPricesCmd, recommendCmd)
	return cmd
}

func NewClusterCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Manage clusters",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all clusters",
		RunE:  runClusterList,
	}

	getCmd := &cobra.Command{
		Use:   "get [cluster-id]",
		Short: "Get cluster details",
		Args:  cobra.ExactArgs(1),
		RunE:  runClusterGet,
	}

	cmd.AddCommand(listCmd, getCmd)
	return cmd
}

func NewConfigCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
	}

	viewCmd := &cobra.Command{
		Use:   "view",
		Short: "Show current configuration",
		RunE:  runConfigView,
	}

	setCmd := &cobra.Command{
		Use:   "set [key] [value]",
		Short: "Set configuration value",
		Args:  cobra.ExactArgs(2),
		RunE:  runConfigSet,
	}

	cmd.AddCommand(viewCmd, setCmd)
	return cmd
}

func NewLoginCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Login to EdgeHub",
		RunE:  runLogin,
	}
}

func doRequest(method, path string, body io.Reader) ([]byte, error) {
	url := serverAddr + "/api/v1" + path
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("request failed: %s", string(data))
	}

	return data, nil
}

func printJSON(v interface{}) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func printTable(headers []string, rows [][]string) {
	for i, h := range headers {
		if i > 0 {
			fmt.Print(" | ")
		}
		fmt.Print(h)
	}
	fmt.Println()
	for _, row := range rows {
		for i, cell := range row {
			if i > 0 {
				fmt.Print(" | ")
			}
			fmt.Print(cell)
		}
		fmt.Println()
	}
}

func runNodeList(cmd *cobra.Command, args []string) error {
	data, err := doRequest("GET", "/nodes", nil)
	if err != nil {
		return err
	}

	if outputJSON {
		printJSON(string(data))
	} else {
		fmt.Println(string(data))
	}
	return nil
}

func runNodeRegister(cmd *cobra.Command, args []string) error {
	fmt.Println("Registering node...")
	return nil
}

func runNodeGet(cmd *cobra.Command, args []string) error {
	nodeID := args[0]
	data, err := doRequest("GET", "/nodes/"+nodeID, nil)
	if err != nil {
		return err
	}

	if outputJSON {
		printJSON(string(data))
	} else {
		fmt.Println(string(data))
	}
	return nil
}

func runNodeDelete(cmd *cobra.Command, args []string) error {
	nodeID := args[0]
	_, err := doRequest("DELETE", "/nodes/"+nodeID, nil)
	return err
}

func runJobList(cmd *cobra.Command, args []string) error {
	data, err := doRequest("GET", "/jobs", nil)
	if err != nil {
		return err
	}

	if outputJSON {
		printJSON(string(data))
	} else {
		fmt.Println(string(data))
	}
	return nil
}

func runJobSubmit(cmd *cobra.Command, args []string) error {
	fmt.Println("Submitting job...")
	return nil
}

func runJobGet(cmd *cobra.Command, args []string) error {
	jobID := args[0]
	data, err := doRequest("GET", "/jobs/"+jobID, nil)
	if err != nil {
		return err
	}

	if outputJSON {
		printJSON(string(data))
	} else {
		fmt.Println(string(data))
	}
	return nil
}

func runJobLogs(cmd *cobra.Command, args []string) error {
	jobID := args[0]
	data, err := doRequest("GET", "/jobs/"+jobID+"/logs", nil)
	if err != nil {
		return err
	}
	fmt.Print(string(data))
	return nil
}

func runJobStop(cmd *cobra.Command, args []string) error {
	jobID := args[0]
	_, err := doRequest("POST", "/jobs/"+jobID+"/stop", nil)
	return err
}

func runJobDelete(cmd *cobra.Command, args []string) error {
	jobID := args[0]
	_, err := doRequest("DELETE", "/jobs/"+jobID, nil)
	return err
}

func runMarketListOffers(cmd *cobra.Command, args []string) error {
	data, err := doRequest("GET", "/market/offers", nil)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func runMarketCreateOffer(cmd *cobra.Command, args []string) error {
	fmt.Println("Creating offer...")
	return nil
}

func runMarketListPrices(cmd *cobra.Command, args []string) error {
	data, err := doRequest("GET", "/market/prices", nil)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func runMarketRecommend(cmd *cobra.Command, args []string) error {
	data, err := doRequest("GET", "/market/prices/recommend", nil)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func runClusterList(cmd *cobra.Command, args []string) error {
	data, err := doRequest("GET", "/clusters", nil)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func runClusterGet(cmd *cobra.Command, args []string) error {
	clusterID := args[0]
	data, err := doRequest("GET", "/clusters/"+clusterID, nil)
	if err != nil {
		return err
	}
	fmt.Println(string(data))
	return nil
}

func runConfigView(cmd *cobra.Command, args []string) error {
	fmt.Printf("Server: %s\n", serverAddr)
	fmt.Printf("API Key: %s\n", maskAPIKey(apiKey))
	return nil
}

func runConfigSet(cmd *cobra.Command, args []string) error {
	key, value := args[0], args[1]
	fmt.Printf("Setting %s = %s\n", key, value)
	return nil
}

func runLogin(cmd *cobra.Command, args []string) error {
	fmt.Println("Login functionality not implemented")
	return nil
}

func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

func exitWithError(msg string) {
	fmt.Fprintf(os.Stderr, "Error: %s\n", msg)
	os.Exit(1)
}
