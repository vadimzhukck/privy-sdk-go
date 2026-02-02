package privy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// MockPrivyServer simulates the Privy API for e2e testing.
type MockPrivyServer struct {
	mu            sync.RWMutex
	users         map[string]*User
	wallets       map[string]*Wallet
	policies      map[string]*Policy
	conditionSets map[string]*ConditionSet
	keyQuorums    map[string]*KeyQuorum
	transactions  map[string]*Transaction
	csItems       map[string]map[string]*ConditionSetItem // conditionSetID -> itemID -> item

	userCounter    int
	walletCounter  int
	policyCounter  int
	csCounter      int
	kqCounter      int
	txCounter      int
	itemCounter    int
}

// NewMockPrivyServer creates a new mock server instance.
func NewMockPrivyServer() *MockPrivyServer {
	return &MockPrivyServer{
		users:         make(map[string]*User),
		wallets:       make(map[string]*Wallet),
		policies:      make(map[string]*Policy),
		conditionSets: make(map[string]*ConditionSet),
		keyQuorums:    make(map[string]*KeyQuorum),
		transactions:  make(map[string]*Transaction),
		csItems:       make(map[string]map[string]*ConditionSetItem),
	}
}

// ServeHTTP handles HTTP requests to the mock server.
func (m *MockPrivyServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Validate authentication headers
	if r.Header.Get("Authorization") == "" {
		m.writeError(w, http.StatusUnauthorized, "Missing Authorization header")
		return
	}
	if r.Header.Get("privy-app-id") == "" {
		m.writeError(w, http.StatusUnauthorized, "Missing privy-app-id header")
		return
	}

	path := r.URL.Path
	method := r.Method

	// Route requests
	switch {
	// Users endpoints
	case path == "/v1/users" && method == "POST":
		m.handleCreateUser(w, r)
	case path == "/api/v1/users" && method == "GET":
		m.handleListUsers(w, r)
	case path == "/api/v1/users/email/address" && method == "POST":
		m.handleGetUserByEmail(w, r)
	case path == "/api/v1/users/phone/number" && method == "POST":
		m.handleGetUserByPhone(w, r)
	case path == "/api/v1/users/wallet/address" && method == "POST":
		m.handleGetUserByWallet(w, r)
	case strings.HasPrefix(path, "/api/v1/users/") && strings.HasSuffix(path, "/custom_metadata") && method == "POST":
		m.handleUpdateUserMetadata(w, r)
	case strings.HasPrefix(path, "/api/v1/users/") && method == "DELETE":
		m.handleDeleteUser(w, r)
	case strings.HasPrefix(path, "/api/v1/users/") && method == "GET":
		m.handleGetUser(w, r)

	// Wallets endpoints
	case path == "/v1/wallets" && method == "POST":
		m.handleCreateWallet(w, r)
	case path == "/v1/wallets" && method == "GET":
		m.handleListWallets(w, r)
	case strings.HasPrefix(path, "/v1/wallets/") && strings.HasSuffix(path, "/rpc") && method == "POST":
		m.handleWalletRPC(w, r)
	case strings.HasPrefix(path, "/v1/wallets/") && strings.HasSuffix(path, "/balance") && method == "GET":
		m.handleGetWalletBalance(w, r)
	case strings.HasPrefix(path, "/v1/wallets/") && strings.HasSuffix(path, "/transactions") && method == "GET":
		m.handleGetWalletTransactions(w, r)
	case strings.HasPrefix(path, "/v1/wallets/") && strings.HasSuffix(path, "/export") && method == "POST":
		m.handleExportWallet(w, r)
	case strings.HasPrefix(path, "/v1/wallets/") && method == "GET":
		m.handleGetWallet(w, r)
	case strings.HasPrefix(path, "/v1/wallets/") && method == "PATCH":
		m.handleUpdateWallet(w, r)
	case path == "/v1/wallets/import/initialize" && method == "POST":
		m.handleInitializeImport(w, r)
	case path == "/v1/wallets/import/submit" && method == "POST":
		m.handleSubmitImport(w, r)

	// Transactions endpoints
	case strings.HasPrefix(path, "/v1/transactions/") && method == "GET":
		m.handleGetTransaction(w, r)

	// Policies endpoints
	case path == "/v1/policies" && method == "POST":
		m.handleCreatePolicy(w, r)
	case strings.HasPrefix(path, "/v1/policies/") && strings.Contains(path, "/rules/") && method == "GET":
		m.handleGetPolicyRule(w, r)
	case strings.HasPrefix(path, "/v1/policies/") && strings.Contains(path, "/rules/") && method == "PATCH":
		m.handleUpdatePolicyRule(w, r)
	case strings.HasPrefix(path, "/v1/policies/") && strings.Contains(path, "/rules/") && method == "DELETE":
		m.handleDeletePolicyRule(w, r)
	case strings.HasPrefix(path, "/v1/policies/") && strings.HasSuffix(path, "/rules") && method == "POST":
		m.handleAddPolicyRule(w, r)
	case strings.HasPrefix(path, "/v1/policies/") && method == "GET":
		m.handleGetPolicy(w, r)
	case strings.HasPrefix(path, "/v1/policies/") && method == "PATCH":
		m.handleUpdatePolicy(w, r)
	case strings.HasPrefix(path, "/v1/policies/") && method == "DELETE":
		m.handleDeletePolicy(w, r)

	// Condition Sets endpoints
	case path == "/v1/condition-sets" && method == "POST":
		m.handleCreateConditionSet(w, r)
	case strings.HasPrefix(path, "/v1/condition-sets/") && strings.Contains(path, "/items/") && method == "GET":
		m.handleGetConditionSetItem(w, r)
	case strings.HasPrefix(path, "/v1/condition-sets/") && strings.Contains(path, "/items/") && method == "DELETE":
		m.handleDeleteConditionSetItem(w, r)
	case strings.HasPrefix(path, "/v1/condition-sets/") && strings.HasSuffix(path, "/items") && method == "POST":
		m.handleAddConditionSetItems(w, r)
	case strings.HasPrefix(path, "/v1/condition-sets/") && strings.HasSuffix(path, "/items") && method == "GET":
		m.handleListConditionSetItems(w, r)
	case strings.HasPrefix(path, "/v1/condition-sets/") && strings.HasSuffix(path, "/items") && method == "PATCH":
		m.handleReplaceConditionSetItems(w, r)
	case strings.HasPrefix(path, "/v1/condition-sets/") && method == "GET":
		m.handleGetConditionSet(w, r)
	case strings.HasPrefix(path, "/v1/condition-sets/") && method == "PATCH":
		m.handleUpdateConditionSet(w, r)
	case strings.HasPrefix(path, "/v1/condition-sets/") && method == "DELETE":
		m.handleDeleteConditionSet(w, r)

	// Key Quorums endpoints
	case path == "/v1/key-quorums" && method == "POST":
		m.handleCreateKeyQuorum(w, r)
	case strings.HasPrefix(path, "/v1/key-quorums/") && method == "GET":
		m.handleGetKeyQuorum(w, r)
	case strings.HasPrefix(path, "/v1/key-quorums/") && method == "PATCH":
		m.handleUpdateKeyQuorum(w, r)
	case strings.HasPrefix(path, "/v1/key-quorums/") && method == "DELETE":
		m.handleDeleteKeyQuorum(w, r)

	default:
		m.writeError(w, http.StatusNotFound, fmt.Sprintf("Unknown endpoint: %s %s", method, path))
	}
}

func (m *MockPrivyServer) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (m *MockPrivyServer) writeError(w http.ResponseWriter, status int, message string) {
	m.writeJSON(w, status, map[string]string{"message": message})
}

// User handlers
func (m *MockPrivyServer) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req CreateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.userCounter++
	userID := fmt.Sprintf("did:privy:user%d", m.userCounter)

	linkedAccounts := make([]LinkedAccount, len(req.LinkedAccounts))
	for i, la := range req.LinkedAccounts {
		linkedAccounts[i] = LinkedAccount{
			Type:            la.Type,
			Address:         la.Address,
			PhoneNumber:     la.PhoneNumber,
			VerifiedAt:      time.Now().UnixMilli(),
			FirstVerifiedAt: time.Now().UnixMilli(),
		}
	}

	user := &User{
		ID:             userID,
		CreatedAt:      time.Now().UnixMilli(),
		LinkedAccounts: linkedAccounts,
		CustomMetadata: req.CustomMetadata,
	}

	// Create embedded wallet if requested
	if req.CreateEthereumWallet {
		m.walletCounter++
		wallet := &Wallet{
			ID:        fmt.Sprintf("wallet-%d", m.walletCounter),
			Address:   fmt.Sprintf("0x%040d", m.walletCounter),
			ChainType: ChainTypeEthereum,
			CreatedAt: time.Now().UnixMilli(),
		}
		m.wallets[wallet.ID] = wallet
		user.LinkedAccounts = append(user.LinkedAccounts, LinkedAccount{
			Type:      LinkedAccountTypeWallet,
			Address:   wallet.Address,
			ChainType: ChainTypeEthereum,
		})
	}

	m.users[userID] = user
	m.writeJSON(w, http.StatusOK, user)
}

func (m *MockPrivyServer) handleGetUser(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	userID := parts[len(parts)-1]

	m.mu.RLock()
	user, exists := m.users[userID]
	m.mu.RUnlock()

	if !exists {
		m.writeError(w, http.StatusNotFound, "User not found")
		return
	}

	m.writeJSON(w, http.StatusOK, user)
}

func (m *MockPrivyServer) handleListUsers(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	users := make([]User, 0, len(m.users))
	for _, u := range m.users {
		users = append(users, *u)
	}
	m.mu.RUnlock()

	resp := PaginatedResponse[User]{
		Data: users,
	}
	m.writeJSON(w, http.StatusOK, resp)
}

func (m *MockPrivyServer) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	userID := parts[len(parts)-1]

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.users[userID]; !exists {
		m.writeError(w, http.StatusNotFound, "User not found")
		return
	}

	delete(m.users, userID)
	w.WriteHeader(http.StatusNoContent)
}

func (m *MockPrivyServer) handleGetUserByEmail(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	email := req["address"]

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, user := range m.users {
		for _, la := range user.LinkedAccounts {
			if la.Type == LinkedAccountTypeEmail && la.Address == email {
				m.writeJSON(w, http.StatusOK, user)
				return
			}
		}
	}

	m.writeError(w, http.StatusNotFound, "User not found")
}

func (m *MockPrivyServer) handleGetUserByPhone(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	phone := req["number"]

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, user := range m.users {
		for _, la := range user.LinkedAccounts {
			if la.Type == LinkedAccountTypePhone && la.PhoneNumber == phone {
				m.writeJSON(w, http.StatusOK, user)
				return
			}
		}
	}

	m.writeError(w, http.StatusNotFound, "User not found")
}

func (m *MockPrivyServer) handleGetUserByWallet(w http.ResponseWriter, r *http.Request) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	address := req["address"]

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, user := range m.users {
		for _, la := range user.LinkedAccounts {
			if la.Type == LinkedAccountTypeWallet && la.Address == address {
				m.writeJSON(w, http.StatusOK, user)
				return
			}
		}
	}

	m.writeError(w, http.StatusNotFound, "User not found")
}

func (m *MockPrivyServer) handleUpdateUserMetadata(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	userID := parts[len(parts)-2]

	var req UpdateUserMetadataRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	user, exists := m.users[userID]
	if !exists {
		m.writeError(w, http.StatusNotFound, "User not found")
		return
	}

	user.CustomMetadata = req.CustomMetadata
	m.writeJSON(w, http.StatusOK, user)
}

// Wallet handlers
func (m *MockPrivyServer) handleCreateWallet(w http.ResponseWriter, r *http.Request) {
	var req CreateWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.walletCounter++
	walletID := fmt.Sprintf("wallet-%d", m.walletCounter)

	var address string
	switch req.ChainType {
	case ChainTypeSolana:
		address = fmt.Sprintf("So1ana%040d", m.walletCounter)
	case ChainTypeEthereum:
		address = fmt.Sprintf("0x%040d", m.walletCounter)
	default:
		address = fmt.Sprintf("addr-%s-%d", req.ChainType, m.walletCounter)
	}

	wallet := &Wallet{
		ID:        walletID,
		Address:   address,
		ChainType: req.ChainType,
		PolicyIDs: req.PolicyIDs,
		CreatedAt: time.Now().UnixMilli(),
	}

	if req.Owner != nil && req.Owner.UserID != "" {
		wallet.OwnerID = req.Owner.UserID
	}

	m.wallets[walletID] = wallet
	m.writeJSON(w, http.StatusOK, wallet)
}

func (m *MockPrivyServer) handleGetWallet(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	walletID := parts[len(parts)-1]

	m.mu.RLock()
	wallet, exists := m.wallets[walletID]
	m.mu.RUnlock()

	if !exists {
		m.writeError(w, http.StatusNotFound, "Wallet not found")
		return
	}

	m.writeJSON(w, http.StatusOK, wallet)
}

func (m *MockPrivyServer) handleListWallets(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	wallets := make([]Wallet, 0, len(m.wallets))
	for _, wal := range m.wallets {
		wallets = append(wallets, *wal)
	}
	m.mu.RUnlock()

	resp := PaginatedResponse[Wallet]{
		Data: wallets,
	}
	m.writeJSON(w, http.StatusOK, resp)
}

func (m *MockPrivyServer) handleUpdateWallet(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	walletID := parts[len(parts)-1]

	var req UpdateWalletRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	wallet, exists := m.wallets[walletID]
	if !exists {
		m.writeError(w, http.StatusNotFound, "Wallet not found")
		return
	}

	if req.PolicyIDs != nil {
		wallet.PolicyIDs = req.PolicyIDs
	}

	m.writeJSON(w, http.StatusOK, wallet)
}

func (m *MockPrivyServer) handleGetWalletBalance(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	walletID := parts[len(parts)-2]

	m.mu.RLock()
	_, exists := m.wallets[walletID]
	m.mu.RUnlock()

	if !exists {
		m.writeError(w, http.StatusNotFound, "Wallet not found")
		return
	}

	balance := &WalletBalance{
		Balance:  "1000000000000000000",
		Currency: "ETH",
		Symbol:   "ETH",
	}
	m.writeJSON(w, http.StatusOK, balance)
}

func (m *MockPrivyServer) handleGetWalletTransactions(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	walletID := parts[len(parts)-2]

	m.mu.RLock()
	_, exists := m.wallets[walletID]
	m.mu.RUnlock()

	if !exists {
		m.writeError(w, http.StatusNotFound, "Wallet not found")
		return
	}

	// Validate required query parameters
	chain := r.URL.Query().Get("chain")
	assets := r.URL.Query()["asset"]

	if chain == "" {
		m.writeError(w, http.StatusBadRequest, "chain parameter is required")
		return
	}

	if len(assets) == 0 {
		m.writeError(w, http.StatusBadRequest, "asset parameter is required")
		return
	}

	resp := PaginatedResponse[Transaction]{
		Data: []Transaction{},
	}
	m.writeJSON(w, http.StatusOK, resp)
}

func (m *MockPrivyServer) handleExportWallet(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	walletID := parts[len(parts)-2]

	m.mu.RLock()
	_, exists := m.wallets[walletID]
	m.mu.RUnlock()

	if !exists {
		m.writeError(w, http.StatusNotFound, "Wallet not found")
		return
	}

	resp := &ExportWalletResponse{
		PrivateKey: "0xprivatekey1234567890abcdef",
	}
	m.writeJSON(w, http.StatusOK, resp)
}

func (m *MockPrivyServer) handleInitializeImport(w http.ResponseWriter, r *http.Request) {
	resp := &ImportWalletInitResponse{
		ImportID:  "import-123",
		PublicKey: "0xpublickey",
	}
	m.writeJSON(w, http.StatusOK, resp)
}

func (m *MockPrivyServer) handleSubmitImport(w http.ResponseWriter, r *http.Request) {
	var req ImportWalletSubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.walletCounter++
	wallet := &Wallet{
		ID:        fmt.Sprintf("wallet-%d", m.walletCounter),
		Address:   fmt.Sprintf("0x%040d", m.walletCounter),
		ChainType: ChainTypeEthereum,
		CreatedAt: time.Now().UnixMilli(),
	}

	m.wallets[wallet.ID] = wallet
	m.writeJSON(w, http.StatusOK, wallet)
}

func (m *MockPrivyServer) handleWalletRPC(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	walletID := parts[len(parts)-2]

	m.mu.RLock()
	wallet, exists := m.wallets[walletID]
	m.mu.RUnlock()

	if !exists {
		m.writeError(w, http.StatusNotFound, "Wallet not found")
		return
	}

	var req RPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	resp := SignatureResponse{
		Method: req.Method,
	}

	switch req.Method {
	case "personal_sign", "signMessage":
		resp.Data.Signature = "0xsignature1234567890"
		resp.Data.Encoding = "hex"
	case "eth_signTransaction", "signTransaction":
		resp.Data.SignedTransaction = "0xsignedtx1234567890"
		resp.Data.Encoding = "rlp"
	case "eth_sendTransaction", "signAndSendTransaction":
		m.mu.Lock()
		m.txCounter++
		txID := fmt.Sprintf("tx-%d", m.txCounter)
		tx := &Transaction{
			ID:        txID,
			WalletID:  wallet.ID,
			ChainType: wallet.ChainType,
			CAIP2:     req.CAIP2,
			Hash:      fmt.Sprintf("0xtxhash%d", m.txCounter),
			Status:    "pending",
			CreatedAt: time.Now().UnixMilli(),
		}
		m.transactions[txID] = tx
		m.mu.Unlock()

		resp.Data.Hash = tx.Hash
		resp.Data.CAIP2 = req.CAIP2
	case "eth_signTypedData_v4":
		resp.Data.Signature = "0xtypeddatasig1234567890"
		resp.Data.Encoding = "hex"
	case "secp256k1_sign", "raw_sign":
		resp.Data.Signature = "0xrawsig1234567890"
		resp.Data.Encoding = "hex"
	case "eth_signUserOperation":
		resp.Data.Signature = "0xuserop1234567890"
		resp.Data.Encoding = "hex"
	case "eth_sign7702Authorization":
		resp.Data.Signature = "0x7702sig1234567890"
		resp.Data.Encoding = "hex"
	default:
		m.writeError(w, http.StatusBadRequest, fmt.Sprintf("Unknown RPC method: %s", req.Method))
		return
	}

	m.writeJSON(w, http.StatusOK, resp)
}

func (m *MockPrivyServer) handleGetTransaction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	txID := parts[len(parts)-1]

	m.mu.RLock()
	tx, exists := m.transactions[txID]
	m.mu.RUnlock()

	if !exists {
		m.writeError(w, http.StatusNotFound, "Transaction not found")
		return
	}

	m.writeJSON(w, http.StatusOK, tx)
}

// Policy handlers
func (m *MockPrivyServer) handleCreatePolicy(w http.ResponseWriter, r *http.Request) {
	var req CreatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.policyCounter++
	policyID := fmt.Sprintf("policy-%d", m.policyCounter)

	policy := &Policy{
		ID:        policyID,
		Name:      req.Name,
		Rules:     req.Rules,
		CreatedAt: time.Now().UnixMilli(),
	}

	m.policies[policyID] = policy
	m.writeJSON(w, http.StatusOK, policy)
}

func (m *MockPrivyServer) handleGetPolicy(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	policyID := parts[len(parts)-1]

	m.mu.RLock()
	policy, exists := m.policies[policyID]
	m.mu.RUnlock()

	if !exists {
		m.writeError(w, http.StatusNotFound, "Policy not found")
		return
	}

	m.writeJSON(w, http.StatusOK, policy)
}

func (m *MockPrivyServer) handleUpdatePolicy(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	policyID := parts[len(parts)-1]

	var req UpdatePolicyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	policy, exists := m.policies[policyID]
	if !exists {
		m.writeError(w, http.StatusNotFound, "Policy not found")
		return
	}

	if req.Name != "" {
		policy.Name = req.Name
	}
	policy.UpdatedAt = time.Now().UnixMilli()

	m.writeJSON(w, http.StatusOK, policy)
}

func (m *MockPrivyServer) handleDeletePolicy(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	policyID := parts[len(parts)-1]

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.policies[policyID]; !exists {
		m.writeError(w, http.StatusNotFound, "Policy not found")
		return
	}

	delete(m.policies, policyID)
	w.WriteHeader(http.StatusNoContent)
}

func (m *MockPrivyServer) handleAddPolicyRule(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	policyID := parts[len(parts)-2]

	var req CreateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	policy, exists := m.policies[policyID]
	if !exists {
		m.writeError(w, http.StatusNotFound, "Policy not found")
		return
	}

	rule := PolicyRule{
		ID:         fmt.Sprintf("rule-%d", len(policy.Rules)+1),
		Action:     req.Action,
		Conditions: req.Conditions,
	}

	policy.Rules = append(policy.Rules, rule)
	m.writeJSON(w, http.StatusOK, rule)
}

func (m *MockPrivyServer) handleGetPolicyRule(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	policyID := parts[len(parts)-3]
	ruleID := parts[len(parts)-1]

	m.mu.RLock()
	defer m.mu.RUnlock()

	policy, exists := m.policies[policyID]
	if !exists {
		m.writeError(w, http.StatusNotFound, "Policy not found")
		return
	}

	for _, rule := range policy.Rules {
		if rule.ID == ruleID {
			m.writeJSON(w, http.StatusOK, rule)
			return
		}
	}

	m.writeError(w, http.StatusNotFound, "Rule not found")
}

func (m *MockPrivyServer) handleUpdatePolicyRule(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	policyID := parts[len(parts)-3]
	ruleID := parts[len(parts)-1]

	var req UpdateRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	policy, exists := m.policies[policyID]
	if !exists {
		m.writeError(w, http.StatusNotFound, "Policy not found")
		return
	}

	for i, rule := range policy.Rules {
		if rule.ID == ruleID {
			if req.Action != "" {
				policy.Rules[i].Action = req.Action
			}
			if req.Conditions != nil {
				policy.Rules[i].Conditions = req.Conditions
			}
			m.writeJSON(w, http.StatusOK, policy.Rules[i])
			return
		}
	}

	m.writeError(w, http.StatusNotFound, "Rule not found")
}

func (m *MockPrivyServer) handleDeletePolicyRule(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	policyID := parts[len(parts)-3]
	ruleID := parts[len(parts)-1]

	m.mu.Lock()
	defer m.mu.Unlock()

	policy, exists := m.policies[policyID]
	if !exists {
		m.writeError(w, http.StatusNotFound, "Policy not found")
		return
	}

	for i, rule := range policy.Rules {
		if rule.ID == ruleID {
			policy.Rules = append(policy.Rules[:i], policy.Rules[i+1:]...)
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}

	m.writeError(w, http.StatusNotFound, "Rule not found")
}

// Condition Set handlers
func (m *MockPrivyServer) handleCreateConditionSet(w http.ResponseWriter, r *http.Request) {
	var req CreateConditionSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.csCounter++
	csID := fmt.Sprintf("cs-%d", m.csCounter)

	cs := &ConditionSet{
		ID:        csID,
		Name:      req.Name,
		OwnerID:   req.OwnerID,
		CreatedAt: time.Now().UnixMilli(),
	}

	m.conditionSets[csID] = cs
	m.csItems[csID] = make(map[string]*ConditionSetItem)
	m.writeJSON(w, http.StatusOK, cs)
}

func (m *MockPrivyServer) handleGetConditionSet(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	csID := parts[len(parts)-1]

	m.mu.RLock()
	cs, exists := m.conditionSets[csID]
	m.mu.RUnlock()

	if !exists {
		m.writeError(w, http.StatusNotFound, "Condition set not found")
		return
	}

	m.writeJSON(w, http.StatusOK, cs)
}

func (m *MockPrivyServer) handleUpdateConditionSet(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	csID := parts[len(parts)-1]

	var req UpdateConditionSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	cs, exists := m.conditionSets[csID]
	if !exists {
		m.writeError(w, http.StatusNotFound, "Condition set not found")
		return
	}

	if req.Name != "" {
		cs.Name = req.Name
	}
	cs.UpdatedAt = time.Now().UnixMilli()

	m.writeJSON(w, http.StatusOK, cs)
}

func (m *MockPrivyServer) handleDeleteConditionSet(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	csID := parts[len(parts)-1]

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.conditionSets[csID]; !exists {
		m.writeError(w, http.StatusNotFound, "Condition set not found")
		return
	}

	delete(m.conditionSets, csID)
	delete(m.csItems, csID)
	w.WriteHeader(http.StatusNoContent)
}

func (m *MockPrivyServer) handleAddConditionSetItems(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	csID := parts[len(parts)-2]

	var req AddConditionSetItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.conditionSets[csID]; !exists {
		m.writeError(w, http.StatusNotFound, "Condition set not found")
		return
	}

	items := make([]ConditionSetItem, len(req.Items))
	for i, item := range req.Items {
		m.itemCounter++
		itemID := fmt.Sprintf("item-%d", m.itemCounter)
		csItem := &ConditionSetItem{
			ID:    itemID,
			Value: item.Value,
		}
		m.csItems[csID][itemID] = csItem
		items[i] = *csItem
	}

	m.writeJSON(w, http.StatusOK, items)
}

func (m *MockPrivyServer) handleListConditionSetItems(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	csID := parts[len(parts)-2]

	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.conditionSets[csID]; !exists {
		m.writeError(w, http.StatusNotFound, "Condition set not found")
		return
	}

	items := make([]ConditionSetItem, 0)
	for _, item := range m.csItems[csID] {
		items = append(items, *item)
	}

	resp := PaginatedResponse[ConditionSetItem]{
		Data: items,
	}
	m.writeJSON(w, http.StatusOK, resp)
}

func (m *MockPrivyServer) handleGetConditionSetItem(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	csID := parts[len(parts)-3]
	itemID := parts[len(parts)-1]

	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.conditionSets[csID]; !exists {
		m.writeError(w, http.StatusNotFound, "Condition set not found")
		return
	}

	item, exists := m.csItems[csID][itemID]
	if !exists {
		m.writeError(w, http.StatusNotFound, "Item not found")
		return
	}

	m.writeJSON(w, http.StatusOK, item)
}

func (m *MockPrivyServer) handleReplaceConditionSetItems(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	csID := parts[len(parts)-2]

	var req ReplaceConditionSetItemsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.conditionSets[csID]; !exists {
		m.writeError(w, http.StatusNotFound, "Condition set not found")
		return
	}

	// Clear existing items
	m.csItems[csID] = make(map[string]*ConditionSetItem)

	items := make([]ConditionSetItem, len(req.Items))
	for i, item := range req.Items {
		m.itemCounter++
		itemID := fmt.Sprintf("item-%d", m.itemCounter)
		csItem := &ConditionSetItem{
			ID:    itemID,
			Value: item.Value,
		}
		m.csItems[csID][itemID] = csItem
		items[i] = *csItem
	}

	m.writeJSON(w, http.StatusOK, items)
}

func (m *MockPrivyServer) handleDeleteConditionSetItem(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	csID := parts[len(parts)-3]
	itemID := parts[len(parts)-1]

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.conditionSets[csID]; !exists {
		m.writeError(w, http.StatusNotFound, "Condition set not found")
		return
	}

	if _, exists := m.csItems[csID][itemID]; !exists {
		m.writeError(w, http.StatusNotFound, "Item not found")
		return
	}

	delete(m.csItems[csID], itemID)
	w.WriteHeader(http.StatusNoContent)
}

// Key Quorum handlers
func (m *MockPrivyServer) handleCreateKeyQuorum(w http.ResponseWriter, r *http.Request) {
	var req CreateKeyQuorumRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.kqCounter++
	kqID := fmt.Sprintf("kq-%d", m.kqCounter)

	kq := &KeyQuorum{
		ID:        kqID,
		PublicKey: req.PublicKey,
		CreatedAt: time.Now().UnixMilli(),
	}

	m.keyQuorums[kqID] = kq
	m.writeJSON(w, http.StatusOK, kq)
}

func (m *MockPrivyServer) handleGetKeyQuorum(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	kqID := parts[len(parts)-1]

	m.mu.RLock()
	kq, exists := m.keyQuorums[kqID]
	m.mu.RUnlock()

	if !exists {
		m.writeError(w, http.StatusNotFound, "Key quorum not found")
		return
	}

	m.writeJSON(w, http.StatusOK, kq)
}

func (m *MockPrivyServer) handleUpdateKeyQuorum(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	kqID := parts[len(parts)-1]

	var req UpdateKeyQuorumRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		m.writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	kq, exists := m.keyQuorums[kqID]
	if !exists {
		m.writeError(w, http.StatusNotFound, "Key quorum not found")
		return
	}

	if req.PublicKey != "" {
		kq.PublicKey = req.PublicKey
	}

	m.writeJSON(w, http.StatusOK, kq)
}

func (m *MockPrivyServer) handleDeleteKeyQuorum(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	kqID := parts[len(parts)-1]

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.keyQuorums[kqID]; !exists {
		m.writeError(w, http.StatusNotFound, "Key quorum not found")
		return
	}

	delete(m.keyQuorums, kqID)
	w.WriteHeader(http.StatusNoContent)
}
