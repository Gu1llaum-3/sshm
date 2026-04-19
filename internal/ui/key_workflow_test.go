package ui

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/Gu1llaum-3/sshm/internal/config"
	keypkg "github.com/Gu1llaum-3/sshm/internal/key"

	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func stubKeyInventoryForUITest(t *testing.T, items []keypkg.InventoryItem, err error) {
	t.Helper()

	original := loadKeyInventoryForUI
	loadKeyInventoryForUI = func(context.Context, string) ([]keypkg.InventoryItem, error) {
		return items, err
	}
	t.Cleanup(func() {
		loadKeyInventoryForUI = original
	})
}

func stubClipboardForUITest(t *testing.T, fn func(string) error) {
	t.Helper()

	original := copyToClipboard
	copyToClipboard = fn
	t.Cleanup(func() {
		copyToClipboard = original
	})
}

func stubRevealPathForUITest(t *testing.T, fn func(string) error) {
	t.Helper()

	original := revealPathInManager
	revealPathInManager = fn
	t.Cleanup(func() {
		revealPathInManager = original
	})
}

func stubAddFormDeployForUITest(t *testing.T, fn func(keypkg.DeployOptions) tea.Cmd) {
	t.Helper()

	original := runAddFormDeploy
	runAddFormDeploy = fn
	t.Cleanup(func() {
		runAddFormDeploy = original
	})
}

func stubEditFormDeployForUITest(t *testing.T, fn func(keypkg.DeployOptions) tea.Cmd) {
	t.Helper()

	original := runEditFormDeploy
	runEditFormDeploy = fn
	t.Cleanup(func() {
		runEditFormDeploy = original
	})
}

func stubAttachHostsForUITest(t *testing.T, fn func(string) ([]config.SSHHost, error)) {
	t.Helper()

	original := loadAttachHostsForUI
	loadAttachHostsForUI = fn
	t.Cleanup(func() {
		loadAttachHostsForUI = original
	})
}

func stubAttachKeyForUITest(t *testing.T, fn func(config.SSHHost, string) tea.Cmd) {
	t.Helper()

	original := runAttachKeyForUI
	runAttachKeyForUI = fn
	t.Cleanup(func() {
		runAttachKeyForUI = original
	})
}

func stubDeployKeyForUITest(t *testing.T, fn func(config.SSHHost, string, string) tea.Cmd) {
	t.Helper()

	original := runDeployKeyForUI
	runDeployKeyForUI = fn
	t.Cleanup(func() {
		runDeployKeyForUI = original
	})
}

func TestAddFormOpensKeyPickerFromIdentityField(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	form := NewAddForm("", NewStyles(80), 80, 24, "")
	form.focused = identityInput
	form.currentTab = tabGeneral

	updated, cmd := form.Update(tea.KeyMsg{Type: tea.KeyCtrlO})
	if updated.keyPicker == nil {
		t.Fatalf("keyPicker = nil, want picker to open")
	}
	if !updated.keyPicker.loading {
		t.Fatalf("loading = false, want async picker load")
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want async load command")
	}
}

func TestAddFormOpensKeyPickerWithEnterFromIdentityField(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	form := NewAddForm("", NewStyles(80), 80, 24, "")
	form.focused = identityInput
	form.currentTab = tabGeneral

	updated, cmd := form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if updated.keyPicker == nil {
		t.Fatalf("keyPicker = nil, want picker to open from Enter")
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want async load command")
	}
}

func TestAddFormSelectKeyAppliesIdentityPath(t *testing.T) {
	form := NewAddForm("", NewStyles(80), 80, 24, "")
	form.keyPicker = &keyPickerModel{}

	updated, cmd := form.Update(keyPickerSelectMsg{path: "/tmp/id_ed25519_selected"})
	if updated.keyPicker != nil {
		t.Fatalf("keyPicker != nil after selection")
	}
	if got := updated.inputs[identityInput].Value(); got != "/tmp/id_ed25519_selected" {
		t.Fatalf("identity value = %q", got)
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want focus update command")
	}
}

func TestAddFormFullPickerSelectionFlow(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	form := NewAddForm("", NewStyles(80), 80, 24, "")
	form.focused = identityInput
	form.currentTab = tabGeneral

	form = openLoadedAddPicker(t, form)

	form, cmd := form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatalf("cmd = nil, want selection command")
	}

	msg := cmd()
	form, _ = form.Update(msg)
	if got := form.inputs[identityInput].Value(); got != "/tmp/id_ed25519_demo" {
		t.Fatalf("identity value = %q", got)
	}
	if form.keyPicker != nil {
		t.Fatalf("keyPicker != nil after full selection flow")
	}
}

func TestAddFormTransitionsFromPickerToCreateAndBack(t *testing.T) {
	form := NewAddForm("", NewStyles(80), 80, 24, "")
	form.keyPicker = &keyPickerModel{}

	updated, cmd := form.Update(keyPickerCreateMsg{})
	if updated.keyPicker != nil {
		t.Fatalf("keyPicker != nil after opening create flow")
	}
	if updated.keyCreate == nil {
		t.Fatalf("keyCreate = nil, want create form")
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want blink command when create form opens")
	}

	updated, cmd = updated.Update(keyCreateFinishedMsg{path: "/tmp/id_ed25519_new"})
	if updated.keyCreate != nil {
		t.Fatalf("keyCreate != nil after completion")
	}
	if got := updated.inputs[identityInput].Value(); got != "/tmp/id_ed25519_new" {
		t.Fatalf("identity value = %q", got)
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want focus update command after completion")
	}
}

func TestAddFormPickerGOpensCreateFlow(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	form := NewAddForm("", NewStyles(80), 80, 24, "")
	form.focused = identityInput
	form.currentTab = tabGeneral

	form = openLoadedAddPicker(t, form)

	form, cmd := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	if cmd == nil {
		t.Fatalf("cmd = nil, want create command")
	}

	msg := cmd()
	form, _ = form.Update(msg)
	if form.keyCreate == nil {
		t.Fatalf("keyCreate = nil, want create form to open")
	}
	if form.keyPicker != nil {
		t.Fatalf("keyPicker != nil after opening create flow")
	}
}

func TestAddFormPickerEscReturnsToForm(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	form := NewAddForm("", NewStyles(80), 80, 24, "")
	form.focused = identityInput
	form.currentTab = tabGeneral

	form = openLoadedAddPicker(t, form)

	form, cmd := form.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("cmd = nil, want cancel command")
	}

	msg := cmd()
	form, _ = form.Update(msg)
	if form.keyPicker != nil {
		t.Fatalf("keyPicker != nil after cancel")
	}
	if form.focused != identityInput {
		t.Fatalf("focused = %d, want identityInput", form.focused)
	}
}

func TestAddFormPickerLeftReturnsToForm(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	form := NewAddForm("", NewStyles(80), 80, 24, "")
	form.focused = identityInput
	form.currentTab = tabGeneral

	form = openLoadedAddPicker(t, form)

	form, cmd := form.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if cmd == nil {
		t.Fatalf("cmd = nil, want cancel command")
	}

	msg := cmd()
	form, _ = form.Update(msg)
	if form.keyPicker != nil {
		t.Fatalf("keyPicker != nil after cancel")
	}
	if form.focused != identityInput {
		t.Fatalf("focused = %d, want identityInput", form.focused)
	}
}

func TestAddFormOpensKeyCreateDirectlyFromIdentityField(t *testing.T) {
	form := NewAddForm("", NewStyles(80), 80, 24, "")
	form.focused = identityInput
	form.currentTab = tabGeneral

	updated, cmd := form.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	if updated.keyCreate == nil {
		t.Fatalf("keyCreate = nil, want create form to open")
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want blink command when create form opens")
	}
}

func TestAddFormCtrlDDeploysAndThenSavesHost(t *testing.T) {
	configFile := t.TempDir() + "/config"
	keyDir := t.TempDir()
	privateKey := keyDir + "/id_ed25519_demo"
	publicKey := privateKey + ".pub"
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(publicKey, []byte("ssh-ed25519 AAAATEST demo\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var gotOpts keypkg.DeployOptions
	stubAddFormDeployForUITest(t, func(opts keypkg.DeployOptions) tea.Cmd {
		gotOpts = opts
		return func() tea.Msg { return addFormDeployFinishedMsg{} }
	})

	form := NewAddForm("", NewStyles(80), 80, 24, configFile)
	form.inputs[nameInput].SetValue("demo")
	form.inputs[hostnameInput].SetValue("203.0.113.10")
	form.inputs[userInput].SetValue("root")
	form.inputs[portInput].SetValue("2222")
	form.inputs[identityInput].SetValue(privateKey)
	form.inputs[proxyJumpInput].SetValue("bastion")
	form.inputs[proxyCommandInput].SetValue("ssh -W %h:%p jump-host")

	updated, cmd := form.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	if cmd == nil {
		t.Fatalf("cmd = nil, want deploy command")
	}
	if gotOpts.Target != "203.0.113.10" || gotOpts.User != "root" || gotOpts.Port != "2222" || gotOpts.ProxyJump != "bastion" || gotOpts.ProxyCommand != "ssh -W %h:%p jump-host" || gotOpts.Identity != privateKey {
		t.Fatalf("deploy opts = %#v", gotOpts)
	}

	updated, cmd = updated.Update(cmd())
	if cmd == nil {
		t.Fatalf("cmd = nil after deploy success, want save command")
	}

	updated, _ = updated.Update(cmd())
	if !updated.success {
		t.Fatalf("success = false, want host save after deploy success")
	}

	content, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	if !strings.Contains(string(content), "Host demo") {
		t.Fatalf("config missing host block:\n%s", string(content))
	}
	if !strings.Contains(string(content), "IdentityFile "+privateKey) {
		t.Fatalf("config missing identity:\n%s", string(content))
	}
}

func TestAddFormCtrlDRequiresExplicitIdentity(t *testing.T) {
	form := NewAddForm("", NewStyles(80), 80, 24, "")
	form.inputs[nameInput].SetValue("demo")
	form.inputs[hostnameInput].SetValue("203.0.113.10")

	updated, cmd := form.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	if cmd != nil {
		t.Fatalf("cmd != nil, want validation failure")
	}
	if !strings.Contains(updated.err, "choose or generate a key before deploy") {
		t.Fatalf("err = %q, want explicit identity guidance", updated.err)
	}
}

func TestEditFormSelectKeyAppliesIdentityPath(t *testing.T) {
	form := newTestEditForm()
	form.keyPicker = &keyPickerModel{}

	updatedModel, cmd := form.Update(keyPickerSelectMsg{path: "/tmp/id_ed25519_edit"})
	updated := updatedModel.(*editFormModel)

	if updated.keyPicker != nil {
		t.Fatalf("keyPicker != nil after selection")
	}
	if got := updated.inputs[3].Value(); got != "/tmp/id_ed25519_edit" {
		t.Fatalf("identity value = %q", got)
	}
	if updated.focusArea != focusAreaProperties || updated.currentTab != 0 || updated.focused != 3 {
		t.Fatalf("focus = (%d,%d,%d), want properties/general/identity", updated.focusArea, updated.currentTab, updated.focused)
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want focus update command")
	}
}

func TestEditFormFullPickerSelectionFlow(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	form := newTestEditForm()
	updated := openLoadedEditPicker(t, form)

	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(*editFormModel)
	if cmd == nil {
		t.Fatalf("cmd = nil, want selection command")
	}

	msg := cmd()
	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(*editFormModel)
	if got := updated.inputs[3].Value(); got != "/tmp/id_ed25519_demo" {
		t.Fatalf("identity value = %q", got)
	}
	if updated.keyPicker != nil {
		t.Fatalf("keyPicker != nil after full selection flow")
	}
}

func TestEditFormOpensKeyPickerWithEnterFromIdentityField(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	form := newTestEditForm()

	updatedModel, cmd := form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(*editFormModel)
	if updated.keyPicker == nil {
		t.Fatalf("keyPicker = nil, want picker to open from Enter")
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want async load command")
	}
}

func TestEditFormOpensKeyCreateDirectlyFromIdentityField(t *testing.T) {
	form := newTestEditForm()

	updatedModel, cmd := form.Update(tea.KeyMsg{Type: tea.KeyCtrlG})
	updated := updatedModel.(*editFormModel)
	if updated.keyCreate == nil {
		t.Fatalf("keyCreate = nil, want create form to open")
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want blink command when create form opens")
	}
}

func TestEditFormCtrlDDeploysAndThenSavesHost(t *testing.T) {
	configFile := t.TempDir() + "/config"
	initialConfig := "Host demo\n  HostName 203.0.113.10\n  User root\n  Port 22\n"
	if err := os.WriteFile(configFile, []byte(initialConfig), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	keyDir := t.TempDir()
	privateKey := keyDir + "/id_ed25519_demo"
	publicKey := privateKey + ".pub"
	if err := os.WriteFile(privateKey, []byte("private"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(publicKey, []byte("ssh-ed25519 AAAATEST demo\n"), 0644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var gotOpts keypkg.DeployOptions
	stubEditFormDeployForUITest(t, func(opts keypkg.DeployOptions) tea.Cmd {
		gotOpts = opts
		return func() tea.Msg { return editFormDeployFinishedMsg{} }
	})

	form, err := NewEditForm("demo", NewStyles(80), 80, 24, configFile)
	if err != nil {
		t.Fatalf("NewEditForm() error = %v", err)
	}
	form.focusArea = focusAreaProperties
	form.currentTab = 0
	form.focused = 3
	form.inputs[0].SetValue("203.0.113.11")
	form.inputs[1].SetValue("ubuntu")
	form.inputs[2].SetValue("2222")
	form.inputs[3].SetValue(privateKey)
	form.inputs[4].SetValue("bastion")
	form.inputs[5].SetValue("ssh -W %h:%p jump-host")

	updatedModel, cmd := form.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	updated := updatedModel.(*editFormModel)
	if cmd == nil {
		t.Fatalf("cmd = nil, want deploy command")
	}
	if gotOpts.Target != "203.0.113.11" || gotOpts.User != "ubuntu" || gotOpts.Port != "2222" || gotOpts.ProxyJump != "bastion" || gotOpts.ProxyCommand != "ssh -W %h:%p jump-host" || gotOpts.Identity != privateKey {
		t.Fatalf("deploy opts = %#v", gotOpts)
	}

	updatedModel, cmd = updated.Update(cmd())
	updated = updatedModel.(*editFormModel)
	if cmd == nil {
		t.Fatalf("cmd = nil after deploy success, want save command")
	}

	msg := cmd()
	submitMsg, ok := msg.(editFormSubmitMsg)
	if !ok {
		t.Fatalf("msg type = %T, want editFormSubmitMsg", msg)
	}
	if submitMsg.err != nil {
		t.Fatalf("submit error = %v", submitMsg.err)
	}

	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(*editFormModel)
	if updated.err != "" {
		t.Fatalf("err = %q, want empty after successful deploy+save", updated.err)
	}

	content, err := os.ReadFile(configFile)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	got := string(content)
	if !strings.Contains(got, "HostName 203.0.113.11") {
		t.Fatalf("config missing updated hostname:\n%s", got)
	}
	if !strings.Contains(got, "User ubuntu") {
		t.Fatalf("config missing updated user:\n%s", got)
	}
	if !strings.Contains(got, "Port 2222") {
		t.Fatalf("config missing updated port:\n%s", got)
	}
	if !strings.Contains(got, "IdentityFile "+privateKey) {
		t.Fatalf("config missing identity:\n%s", got)
	}
}

func TestEditFormCtrlDRequiresExplicitIdentity(t *testing.T) {
	form := newTestEditForm()
	form.inputs[0].SetValue("203.0.113.10")

	updatedModel, cmd := form.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	updated := updatedModel.(*editFormModel)
	if cmd != nil {
		t.Fatalf("cmd != nil, want validation failure")
	}
	if !strings.Contains(updated.err, "choose or generate a key before deploy") {
		t.Fatalf("err = %q, want explicit identity guidance", updated.err)
	}
}

func TestEditFormCtrlDDeletesFocusedAliasInHostNamesArea(t *testing.T) {
	form := newTestEditForm()
	form.hostInputs = []textinput.Model{textinput.New(), textinput.New()}
	form.hostInputs[0].SetValue("demo")
	form.hostInputs[1].SetValue("demo-alt")
	form.focusArea = focusAreaHosts
	form.focused = 1

	updatedModel, cmd := form.Update(tea.KeyMsg{Type: tea.KeyCtrlD})
	updated := updatedModel.(*editFormModel)
	if cmd != nil {
		t.Fatalf("cmd != nil, want in-place delete only")
	}
	if len(updated.hostInputs) != 1 {
		t.Fatalf("len(hostInputs) = %d, want 1", len(updated.hostInputs))
	}
	if got := updated.hostInputs[0].Value(); got != "demo" {
		t.Fatalf("remaining host = %q, want demo", got)
	}
}

func TestEditFormPickerGOpensCreateFlow(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	form := newTestEditForm()
	updated := openLoadedEditPicker(t, form)

	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	updated = updatedModel.(*editFormModel)
	if cmd == nil {
		t.Fatalf("cmd = nil, want create command")
	}

	msg := cmd()
	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(*editFormModel)
	if updated.keyCreate == nil {
		t.Fatalf("keyCreate = nil, want create form to open")
	}
	if updated.keyPicker != nil {
		t.Fatalf("keyPicker != nil after opening create flow")
	}
}

func TestEditFormPickerEscReturnsToForm(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	form := newTestEditForm()
	updated := openLoadedEditPicker(t, form)

	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(*editFormModel)
	if cmd == nil {
		t.Fatalf("cmd = nil, want cancel command")
	}

	msg := cmd()
	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(*editFormModel)
	if updated.keyPicker != nil {
		t.Fatalf("keyPicker != nil after cancel")
	}
	if updated.focused != 3 {
		t.Fatalf("focused = %d, want identity field", updated.focused)
	}
}

func TestEditFormPickerLeftReturnsToForm(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	form := newTestEditForm()
	updated := openLoadedEditPicker(t, form)

	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyLeft})
	updated = updatedModel.(*editFormModel)
	if cmd == nil {
		t.Fatalf("cmd = nil, want cancel command")
	}

	msg := cmd()
	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(*editFormModel)
	if updated.keyPicker != nil {
		t.Fatalf("keyPicker != nil after cancel")
	}
	if updated.focused != 3 {
		t.Fatalf("focused = %d, want identity field", updated.focused)
	}
}

func TestKeyCreateFormRequiresName(t *testing.T) {
	form := NewKeyCreateForm(NewStyles(80), 80, 24)
	form.inputs[keyCreateNameInput].SetValue("")

	cmd := form.submit()
	if cmd != nil {
		t.Fatalf("cmd != nil, want validation failure before spawning process")
	}
	if form.err == "" {
		t.Fatalf("err is empty, want validation error")
	}
}

func TestKeyCreateFormAllowsLetterQInput(t *testing.T) {
	form := NewKeyCreateForm(NewStyles(80), 80, 24)

	updated, _ := form.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if got := updated.inputs[keyCreateNameInput].Value(); got != "q" {
		t.Fatalf("name value = %q, want q", got)
	}
	if updated.err != "" {
		t.Fatalf("err = %q, want empty", updated.err)
	}
}

func TestKeyCreateFormRejectsAlgorithmOutsideAllowlist(t *testing.T) {
	form := NewKeyCreateForm(NewStyles(80), 80, 24)
	form.inputs[keyCreateNameInput].SetValue("id_demo")
	form.algorithmOptions = append(form.algorithmOptions, "dsa")
	form.algorithmIndex = len(form.algorithmOptions) - 1

	cmd := form.submit()
	if cmd != nil {
		t.Fatalf("cmd != nil, want validation failure")
	}
	if !strings.Contains(form.err, "algorithm must be one of") {
		t.Fatalf("err = %q, want allowlist validation", form.err)
	}
}

func TestKeyCreateFormAcceptsAllowedAlgorithm(t *testing.T) {
	form := NewKeyCreateForm(NewStyles(80), 80, 24)
	form.inputs[keyCreateNameInput].SetValue("id_demo")
	form.inputs[keyCreatePathInput].SetValue(t.TempDir())
	form.algorithmIndex = 2

	cmd := form.submit()
	if cmd == nil {
		t.Fatalf("cmd = nil, want ssh-keygen process command")
	}
	if form.err != "" {
		t.Fatalf("err = %q, want empty", form.err)
	}
}

func TestKeyCreateFormDownFromGeneralSwitchesToAdvanced(t *testing.T) {
	form := NewKeyCreateForm(NewStyles(80), 80, 24)
	form.focused = keyCreatePathInput
	form.currentTab = keyCreateTabGeneral

	updated, _ := form.Update(tea.KeyMsg{Type: tea.KeyDown})
	if updated.currentTab != keyCreateTabAdvanced {
		t.Fatalf("currentTab = %d, want advanced", updated.currentTab)
	}
	if updated.focused != keyCreateAlgorithmInput {
		t.Fatalf("focused = %d, want algorithm picker", updated.focused)
	}
}

func TestKeyCreateFormUpFromAdvancedReturnsToGeneral(t *testing.T) {
	form := NewKeyCreateForm(NewStyles(80), 80, 24)
	form.focused = keyCreateAlgorithmInput
	form.currentTab = keyCreateTabAdvanced

	updated, _ := form.Update(tea.KeyMsg{Type: tea.KeyUp})
	if updated.currentTab != keyCreateTabGeneral {
		t.Fatalf("currentTab = %d, want general", updated.currentTab)
	}
	if updated.focused != keyCreatePathInput {
		t.Fatalf("focused = %d, want directory field", updated.focused)
	}
}

func TestKeyCreateFormRightCyclesAlgorithmPicker(t *testing.T) {
	form := NewKeyCreateForm(NewStyles(80), 80, 24)
	form.focused = keyCreateAlgorithmInput
	form.currentTab = keyCreateTabAdvanced

	updated, _ := form.Update(tea.KeyMsg{Type: tea.KeyRight})
	if got := updated.algorithmOptions[updated.algorithmIndex]; got != "ecdsa" {
		t.Fatalf("algorithm = %q, want ecdsa", got)
	}
}

func TestRootAddViewRoutesKeyPickerSelectionMessage(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	m := newKeyWorkflowRootModel()
	m.viewMode = ViewAdd
	m.addForm = NewAddForm("", NewStyles(80), 80, 24, "")
	m.addForm.focused = identityInput
	m.addForm.currentTab = tabGeneral

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(Model)
	if updated.addForm == nil || updated.addForm.keyPicker == nil {
		t.Fatalf("keyPicker = nil, want picker to open")
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want async load command")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.addForm.keyPicker.loading {
		t.Fatalf("loading = true after load command")
	}

	updatedModel, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want chooser callback command")
	}

	msg := cmd()
	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(Model)
	if got := updated.addForm.inputs[identityInput].Value(); got != "/tmp/id_ed25519_demo" {
		t.Fatalf("identity value = %q, want selected key path", got)
	}
	if updated.addForm.keyPicker != nil {
		t.Fatalf("keyPicker != nil after routing chooser callback through root")
	}
}

func TestRootAddViewRoutesKeyPickerCancelMessage(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	m := newKeyWorkflowRootModel()
	m.viewMode = ViewAdd
	m.addForm = NewAddForm("", NewStyles(80), 80, 24, "")
	m.addForm.focused = identityInput
	m.addForm.currentTab = tabGeneral

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(Model)
	if updated.addForm == nil || updated.addForm.keyPicker == nil {
		t.Fatalf("keyPicker = nil, want picker to open")
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want async load command")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)

	updatedModel, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want cancel callback command")
	}

	msg := cmd()
	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(Model)
	if updated.addForm.keyPicker != nil {
		t.Fatalf("keyPicker != nil after cancel routed through root")
	}
	if updated.addForm.focused != identityInput {
		t.Fatalf("focused = %d, want identityInput", updated.addForm.focused)
	}
}

func TestRootAddViewRoutesKeyPickerCreateMessage(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	m := newKeyWorkflowRootModel()
	m.viewMode = ViewAdd
	m.addForm = NewAddForm("", NewStyles(80), 80, 24, "")
	m.addForm.focused = identityInput
	m.addForm.currentTab = tabGeneral

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(Model)
	if updated.addForm == nil || updated.addForm.keyPicker == nil {
		t.Fatalf("keyPicker = nil, want picker to open")
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want async load command")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)

	updatedModel, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want create callback command")
	}

	msg := cmd()
	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(Model)
	if updated.addForm.keyCreate == nil {
		t.Fatalf("keyCreate = nil after routing create callback through root")
	}
	if updated.addForm.keyPicker != nil {
		t.Fatalf("keyPicker != nil after switching to create modal")
	}
}

func TestRootEditViewRoutesKeyPickerSelectionMessage(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	m := newKeyWorkflowRootModel()
	m.viewMode = ViewEdit
	m.editForm = newTestEditForm()

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(Model)
	if updated.editForm == nil || updated.editForm.keyPicker == nil {
		t.Fatalf("keyPicker = nil, want picker to open")
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want async load command")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)

	updatedModel, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want chooser callback command")
	}

	msg := cmd()
	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(Model)
	if got := updated.editForm.inputs[3].Value(); got != "/tmp/id_ed25519_demo" {
		t.Fatalf("identity value = %q, want selected key path", got)
	}
	if updated.editForm.keyPicker != nil {
		t.Fatalf("keyPicker != nil after routing chooser callback through root")
	}
}

func TestRootListOpensKeysView(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	m := newKeyWorkflowRootModel()

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")})
	updated := updatedModel.(Model)
	if updated.viewMode != ViewKeys {
		t.Fatalf("viewMode = %d, want ViewKeys", updated.viewMode)
	}
	if updated.keysView == nil {
		t.Fatalf("keysView = nil, want keys browser")
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want async inventory load")
	}
}

func TestRootKeysViewEscClosesToList(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	m := newKeyWorkflowRootModel()

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")})
	updated := updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want async inventory load")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.keysView == nil || updated.keysView.browser == nil || updated.keysView.browser.loading {
		t.Fatalf("keysView still loading after inventory refresh")
	}

	updatedModel, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want close callback")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.viewMode != ViewList {
		t.Fatalf("viewMode = %d, want ViewList", updated.viewMode)
	}
	if updated.keysView != nil {
		t.Fatalf("keysView != nil after close")
	}
}

func TestRootKeysViewLeftClosesToList(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	m := newKeyWorkflowRootModel()

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")})
	updated := updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want async inventory load")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.keysView == nil || updated.keysView.browser == nil || updated.keysView.browser.loading {
		t.Fatalf("keysView still loading after inventory refresh")
	}

	updatedModel, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyLeft})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want close callback")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.viewMode != ViewList {
		t.Fatalf("viewMode = %d, want ViewList", updated.viewMode)
	}
	if updated.keysView != nil {
		t.Fatalf("keysView != nil after close")
	}
}

func TestRootKeysViewGOpensCreateFlow(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	m := newKeyWorkflowRootModel()

	updatedModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")})
	updated := updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want async inventory load")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)

	updatedModel, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("g")})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want create callback")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.keysView == nil || updated.keysView.keyCreate == nil {
		t.Fatalf("keyCreate = nil, want create form inside keys view")
	}
}

func TestRootKeysViewEnterOpensInspect(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}, nil)

	m := newKeyWorkflowRootModel()
	updated := openLoadedKeysView(t, m)

	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	if updated.keysView == nil || !updated.keysView.inspect {
		t.Fatalf("inspect = false, want inspect mode")
	}
}

func TestRootKeysViewInspectLoadsPublicKey(t *testing.T) {
	tempDir := t.TempDir()
	publicKeyPath := tempDir + "/id_ed25519_demo.pub"
	if err := os.WriteFile(publicKeyPath, []byte("ssh-ed25519 AAAATEST demo\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{
			Path:          tempDir + "/id_ed25519_demo",
			PublicKeyPath: publicKeyPath,
			Fingerprint:   "SHA256:test",
			Algorithm:     "ED25519",
		},
	}, nil)

	m := newKeyWorkflowRootModel()
	updated := openLoadedKeysView(t, m)

	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("p")})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want load public key command")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if !updated.keysView.showPublic {
		t.Fatalf("showPublic = false, want visible public key")
	}
	if got := updated.keysView.publicKey; !strings.Contains(got, "AAAATEST") {
		t.Fatalf("publicKey = %q, want key contents", got)
	}
}

func TestRootKeysViewInspectCopiesPublicKey(t *testing.T) {
	tempDir := t.TempDir()
	publicKeyPath := tempDir + "/id_ed25519_demo.pub"
	if err := os.WriteFile(publicKeyPath, []byte("ssh-ed25519 AAAATEST demo\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var copied string
	stubClipboardForUITest(t, func(text string) error {
		copied = text
		return nil
	})
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{
			Path:          tempDir + "/id_ed25519_demo",
			PublicKeyPath: publicKeyPath,
			Fingerprint:   "SHA256:test",
			Algorithm:     "ED25519",
		},
	}, nil)

	m := newKeyWorkflowRootModel()
	updated := openLoadedKeysView(t, m)

	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("c")})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want clipboard command")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if !strings.Contains(copied, "AAAATEST") {
		t.Fatalf("copied = %q, want key contents", copied)
	}
	if got := updated.keysView.status; !strings.Contains(got, "Copied public key.") {
		t.Fatalf("status = %q, want copy status", got)
	}
}

func TestRootKeysViewInspectCopiesKeyPath(t *testing.T) {
	tempDir := t.TempDir()
	privateKeyPath := tempDir + "/id_ed25519_demo"

	var copied string
	stubClipboardForUITest(t, func(text string) error {
		copied = text
		return nil
	})
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{
			Path:        privateKeyPath,
			Fingerprint: "SHA256:test",
			Algorithm:   "ED25519",
		},
	}, nil)

	m := newKeyWorkflowRootModel()
	updated := openLoadedKeysView(t, m)

	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("y")})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want clipboard command")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if copied != privateKeyPath {
		t.Fatalf("copied = %q, want %q", copied, privateKeyPath)
	}
	if got := updated.keysView.status; !strings.Contains(got, "Copied key path.") {
		t.Fatalf("status = %q, want path copy status", got)
	}
}

func TestRootKeysViewInspectRevealsKeyPath(t *testing.T) {
	tempDir := t.TempDir()
	privateKeyPath := tempDir + "/id_ed25519_demo"

	var revealed string
	stubRevealPathForUITest(t, func(path string) error {
		revealed = path
		return nil
	})
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{
			Path:        privateKeyPath,
			Fingerprint: "SHA256:test",
			Algorithm:   "ED25519",
		},
	}, nil)

	m := newKeyWorkflowRootModel()
	updated := openLoadedKeysView(t, m)

	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("o")})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want reveal command")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if revealed != privateKeyPath {
		t.Fatalf("revealed = %q, want %q", revealed, privateKeyPath)
	}
	if got := updated.keysView.status; !strings.Contains(got, "Revealed id_ed25519_demo in file manager.") {
		t.Fatalf("status = %q, want reveal status", got)
	}
}

func TestRootKeysViewInspectTOpensAttachHostPicker(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{
			Path:        "/tmp/id_ed25519_demo",
			Fingerprint: "SHA256:test",
			Algorithm:   "ED25519",
		},
	}, nil)
	stubAttachHostsForUITest(t, func(string) ([]config.SSHHost, error) {
		return []config.SSHHost{{
			Name:       "dev.tisit.vm",
			Hostname:   "203.0.113.10",
			SourceFile: "/tmp/config",
			LineNumber: 1,
		}}, nil
	})

	m := newKeyWorkflowRootModel()
	updated := openLoadedKeysView(t, m)

	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	updated = updatedModel.(Model)
	if updated.keysView == nil || updated.keysView.attachPicker == nil {
		t.Fatalf("attachPicker = nil, want host picker")
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want host load command")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.keysView.attachPicker.loading {
		t.Fatalf("attach picker still loading after callback")
	}
}

func TestRootKeysViewAttachPickerEscReturnsToInspect(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{
			Path:        "/tmp/id_ed25519_demo",
			Fingerprint: "SHA256:test",
			Algorithm:   "ED25519",
		},
	}, nil)
	stubAttachHostsForUITest(t, func(string) ([]config.SSHHost, error) {
		return []config.SSHHost{{
			Name:       "dev.tisit.vm",
			Hostname:   "203.0.113.10",
			SourceFile: "/tmp/config",
			LineNumber: 1,
		}}, nil
	})

	m := newKeyWorkflowRootModel()
	updated := openLoadedKeysView(t, m)
	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	updated = updatedModel.(Model)
	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)

	updatedModel, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want cancel callback")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.keysView.attachPicker != nil {
		t.Fatalf("attachPicker != nil after cancel")
	}
	if updated.keysView == nil || !updated.keysView.inspect {
		t.Fatalf("inspect = false, want return to inspect")
	}
}

func TestRootKeysViewAttachPickerEnterAttachesAndRefreshes(t *testing.T) {
	originalInventoryLoader := loadKeyInventoryForUI
	var inventoryCalls int
	loadKeyInventoryForUI = func(context.Context, string) ([]keypkg.InventoryItem, error) {
		inventoryCalls++
		if inventoryCalls == 1 {
			return []keypkg.InventoryItem{{
				Path:        "/tmp/id_ed25519_demo",
				Fingerprint: "SHA256:test",
				Algorithm:   "ED25519",
			}}, nil
		}
		return []keypkg.InventoryItem{{
			Path:        "/tmp/id_ed25519_demo",
			Fingerprint: "SHA256:test",
			Algorithm:   "ED25519",
			References: []keypkg.Reference{
				{Host: "dev.tisit.vm"},
			},
		}}, nil
	}
	t.Cleanup(func() {
		loadKeyInventoryForUI = originalInventoryLoader
	})

	stubAttachHostsForUITest(t, func(string) ([]config.SSHHost, error) {
		return []config.SSHHost{{
			Name:       "dev.tisit.vm",
			Hostname:   "203.0.113.10",
			SourceFile: "/tmp/config",
			LineNumber: 1,
		}}, nil
	})

	var attachedHost config.SSHHost
	var attachedPath string
	stubAttachKeyForUITest(t, func(host config.SSHHost, path string) tea.Cmd {
		attachedHost = host
		attachedPath = path
		return func() tea.Msg {
			return keyAttachFinishedMsg{hostName: host.Name, path: path}
		}
	})

	m := newKeyWorkflowRootModel()
	updated := openLoadedKeysView(t, m)
	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("t")})
	updated = updatedModel.(Model)
	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)

	updatedModel, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want attach host selection callback")
	}

	updatedModel, cmd = updated.Update(cmd())
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want attach command")
	}
	if attachedHost.Name != "dev.tisit.vm" {
		t.Fatalf("attachedHost = %#v", attachedHost)
	}
	if attachedPath != "/tmp/id_ed25519_demo" {
		t.Fatalf("attachedPath = %q, want selected key path", attachedPath)
	}

	updatedModel, cmd = updated.Update(cmd())
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want inventory refresh")
	}
	if updated.keysView.attachPicker != nil {
		t.Fatalf("attachPicker != nil after attach success")
	}
	if got := updated.keysView.status; !strings.Contains(got, "Attached id_ed25519_demo to dev.tisit.vm.") {
		t.Fatalf("status = %q, want attach status", got)
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if refs := updated.keysView.browser.items[0].References; len(refs) != 1 || refs[0].Host != "dev.tisit.vm" {
		t.Fatalf("references = %#v, want refreshed attach reference", refs)
	}
}

func TestRootKeysViewInspectVOpensDeployHostPicker(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{
			Path:        "/tmp/id_ed25519_demo",
			Fingerprint: "SHA256:test",
			Algorithm:   "ED25519",
		},
	}, nil)
	stubAttachHostsForUITest(t, func(string) ([]config.SSHHost, error) {
		return []config.SSHHost{{
			Name:       "dev.tisit.vm",
			Hostname:   "203.0.113.10",
			SourceFile: "/tmp/config",
			LineNumber: 1,
		}}, nil
	})

	m := newKeyWorkflowRootModel()
	updated := openLoadedKeysView(t, m)

	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	updated = updatedModel.(Model)
	if updated.keysView == nil || updated.keysView.deployPicker == nil {
		t.Fatalf("deployPicker = nil, want host picker")
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want host load command")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.keysView.deployPicker.loading {
		t.Fatalf("deploy picker still loading after callback")
	}
}

func TestRootKeysViewDeployPickerEnterDeploysKey(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{
			Path:        "/tmp/id_ed25519_demo",
			Fingerprint: "SHA256:test",
			Algorithm:   "ED25519",
		},
	}, nil)
	stubAttachHostsForUITest(t, func(string) ([]config.SSHHost, error) {
		return []config.SSHHost{{
			Name:         "dev.tisit.vm",
			Hostname:     "203.0.113.10",
			User:         "ubuntu",
			Port:         "2222",
			ProxyJump:    "bastion",
			ProxyCommand: "ssh -W %h:%p jump-host",
			SourceFile:   "/tmp/config",
			LineNumber:   1,
		}}, nil
	})

	var deployedHost config.SSHHost
	var deployedPath string
	var deployedConfigFile string
	stubDeployKeyForUITest(t, func(host config.SSHHost, path string, configFile string) tea.Cmd {
		deployedHost = host
		deployedPath = path
		deployedConfigFile = configFile
		return func() tea.Msg {
			return keyDeployFinishedMsg{hostName: host.Name, path: path}
		}
	})

	m := newKeyWorkflowRootModel()
	m.configFile = "/tmp/custom-config"
	updated := openLoadedKeysView(t, m)
	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	updated = updatedModel.(Model)
	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)

	updatedModel, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want deploy host selection callback")
	}

	updatedModel, cmd = updated.Update(cmd())
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want deploy command")
	}
	if deployedHost.Name != "dev.tisit.vm" {
		t.Fatalf("deployedHost = %#v", deployedHost)
	}
	if deployedPath != "/tmp/id_ed25519_demo" {
		t.Fatalf("deployedPath = %q, want selected key path", deployedPath)
	}
	if deployedConfigFile != "/tmp/custom-config" {
		t.Fatalf("deployedConfigFile = %q, want custom config", deployedConfigFile)
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.keysView.deployPicker != nil {
		t.Fatalf("deployPicker != nil after deploy success")
	}
	if got := updated.keysView.status; !strings.Contains(got, "Deployed id_ed25519_demo to dev.tisit.vm.") {
		t.Fatalf("status = %q, want deploy status", got)
	}
}

func TestRootKeysViewInspectDeleteBlockedWhenDeclaredRefsExist(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{
			Path:        "/tmp/id_ed25519_demo",
			Fingerprint: "SHA256:test",
			Algorithm:   "ED25519",
			References: []keypkg.Reference{
				{Host: "dev.tisit.vm"},
			},
		},
	}, nil)

	m := newKeyWorkflowRootModel()
	updated := openLoadedKeysView(t, m)

	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updated = updatedModel.(Model)
	if updated.keysView == nil || !updated.keysView.deleteDialog || !updated.keysView.deleteBlocked {
		t.Fatalf("delete dialog not blocked as expected")
	}
}

func TestRootKeysViewInspectDeleteBlockedUsesExplicitAttachCopy(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{
			Path:        "/tmp/id_ed25519_demo",
			Fingerprint: "SHA256:test",
			Algorithm:   "ED25519",
			References: []keypkg.Reference{
				{Host: "dev.tisit.vm"},
				{Host: "github.com"},
			},
		},
	}, nil)

	m := newKeyWorkflowRootModel()
	updated := openLoadedKeysView(t, m)

	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updated = updatedModel.(Model)

	view := updated.keysView.View()
	if !strings.Contains(view, "This key is explicitly referenced by these SSH hosts:") {
		t.Fatalf("blocking copy prefix missing:\n%s", view)
	}
	if !strings.Contains(view, "dev.tisit.vm") || !strings.Contains(view, "github.com") {
		t.Fatalf("blocked hosts missing:\n%s", view)
	}
}

func TestRootKeysViewInspectDeleteConfirmWarnsAboutUnknownImplicitUsage(t *testing.T) {
	privateKeyPath := "/tmp/id_ed25519_demo"
	publicKeyPath := privateKeyPath + ".pub"

	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{
			Path:          privateKeyPath,
			PublicKeyPath: publicKeyPath,
			Fingerprint:   "SHA256:test",
			Algorithm:     "ED25519",
			CanDelete:     true,
		},
	}, nil)

	m := newKeyWorkflowRootModel()
	updated := openLoadedKeysView(t, m)

	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updated = updatedModel.(Model)

	view := updated.keysView.View()
	if !strings.Contains(view, "Delete key") {
		t.Fatalf("delete heading missing:\n%s", view)
	}
	if !strings.Contains(view, displayPath(privateKeyPath)) {
		t.Fatalf("delete title missing:\n%s", view)
	}
	if !strings.Contains(view, "No explicit IdentityFile entries point to this key.") {
		t.Fatalf("explicit refs warning missing:\n%s", view)
	}
	if !strings.Contains(view, "SSHM can't verify inherited or external SSH config.") {
		t.Fatalf("implicit usage warning prefix missing:\n%s", view)
	}
}

func TestRootKeysViewInspectDeleteRemovesFiles(t *testing.T) {
	tempDir := t.TempDir()
	privateKeyPath := tempDir + "/id_ed25519_demo"
	publicKeyPath := privateKeyPath + ".pub"
	if err := os.WriteFile(privateKeyPath, []byte("private"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(publicKeyPath, []byte("public"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{
			Path:          privateKeyPath,
			PublicKeyPath: publicKeyPath,
			Fingerprint:   "SHA256:test",
			Algorithm:     "ED25519",
			CanDelete:     true,
		},
	}, nil)

	m := newKeyWorkflowRootModel()
	updated := openLoadedKeysView(t, m)

	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("d")})
	updated = updatedModel.(Model)
	if updated.keysView == nil || !updated.keysView.deleteDialog || updated.keysView.deleteBlocked {
		t.Fatalf("delete dialog = %+v, want confirm dialog", updated.keysView)
	}

	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want delete command")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if _, err := os.Stat(privateKeyPath); !os.IsNotExist(err) {
		t.Fatalf("private key still exists, err = %v", err)
	}
	if _, err := os.Stat(publicKeyPath); !os.IsNotExist(err) {
		t.Fatalf("public key still exists, err = %v", err)
	}
	if updated.keysView == nil || updated.keysView.inspect {
		t.Fatalf("inspect = true, want return to list after delete")
	}
}

func TestRootKeysViewInspectHOpensLinkedHosts(t *testing.T) {
	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{
			Path:        "/tmp/id_ed25519_demo",
			Fingerprint: "SHA256:test",
			Algorithm:   "ED25519",
			References: []keypkg.Reference{
				{Host: "dev.tisit.vm"},
				{Host: "github.com"},
			},
		},
	}, nil)

	m := newKeyWorkflowRootModel()
	updated := openLoadedKeysView(t, m)

	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	updated = updatedModel.(Model)
	if updated.keysView == nil || !updated.keysView.hostRefs {
		t.Fatalf("hostRefs = false, want linked hosts view")
	}
}

func TestRootKeysViewLinkedHostsEnterOpensHostInfo(t *testing.T) {
	configFile := t.TempDir() + "/config"
	if err := os.WriteFile(configFile, []byte("Host dev.tisit.vm\n  HostName 127.0.0.1\n  User root\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{
			Path:        "/tmp/id_ed25519_demo",
			Fingerprint: "SHA256:test",
			Algorithm:   "ED25519",
			References: []keypkg.Reference{
				{Host: "dev.tisit.vm"},
			},
		},
	}, nil)

	m := newKeyWorkflowRootModel()
	m.configFile = configFile
	updated := openLoadedKeysView(t, m)

	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	updated = updatedModel.(Model)

	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want open host info callback")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.viewMode != ViewInfo {
		t.Fatalf("viewMode = %d, want ViewInfo", updated.viewMode)
	}
	if updated.infoForm == nil || updated.infoForm.hostName != "dev.tisit.vm" {
		t.Fatalf("infoForm host = %#v, want dev.tisit.vm", updated.infoForm)
	}
}

func TestRootKeysViewLinkedHostsUsesReferenceSourceFileForHostInfo(t *testing.T) {
	tempDir := t.TempDir()
	mainConfig := tempDir + "/config"
	includedConfig := tempDir + "/included.conf"
	if err := os.WriteFile(mainConfig, []byte("Host shared\n  HostName main.example.com\nInclude included.conf\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(includedConfig, []byte("Host shared\n  HostName included.example.com\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{
			Path:        "/tmp/id_ed25519_demo",
			Fingerprint: "SHA256:test",
			Algorithm:   "ED25519",
			References: []keypkg.Reference{
				{Host: "shared", SourceFile: includedConfig},
			},
		},
	}, nil)

	m := newKeyWorkflowRootModel()
	m.configFile = mainConfig
	updated := openLoadedKeysView(t, m)

	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	updated = updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want open host info callback")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.infoForm == nil || updated.infoForm.host.Hostname != "included.example.com" {
		t.Fatalf("info host = %#v, want included host", updated.infoForm)
	}
}

func TestRootKeysViewLinkedHostsEOpensHostEdit(t *testing.T) {
	configFile := t.TempDir() + "/config"
	if err := os.WriteFile(configFile, []byte("Host dev.tisit.vm\n  HostName 127.0.0.1\n  User root\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{
			Path:        "/tmp/id_ed25519_demo",
			Fingerprint: "SHA256:test",
			Algorithm:   "ED25519",
			References: []keypkg.Reference{
				{Host: "dev.tisit.vm"},
			},
		},
	}, nil)

	m := newKeyWorkflowRootModel()
	m.configFile = configFile
	updated := openLoadedKeysView(t, m)

	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	updated = updatedModel.(Model)

	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want open host edit callback")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.viewMode != ViewEdit {
		t.Fatalf("viewMode = %d, want ViewEdit", updated.viewMode)
	}
	if updated.editForm == nil {
		t.Fatalf("editForm = nil, want edit form")
	}
}

func TestRootKeysHostInfoEscReturnsToLinkedHosts(t *testing.T) {
	configFile := t.TempDir() + "/config"
	if err := os.WriteFile(configFile, []byte("Host dev.tisit.vm\n  HostName 127.0.0.1\n  User root\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{
			Path:        "/tmp/id_ed25519_demo",
			Fingerprint: "SHA256:test",
			Algorithm:   "ED25519",
			References: []keypkg.Reference{
				{Host: "dev.tisit.vm"},
			},
		},
	}, nil)

	m := newKeyWorkflowRootModel()
	m.configFile = configFile
	updated := openLoadedKeysView(t, m)

	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	updated = updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want open host info callback")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.viewMode != ViewInfo {
		t.Fatalf("viewMode = %d, want ViewInfo", updated.viewMode)
	}

	updatedModel, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want cancel callback")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.viewMode != ViewKeys {
		t.Fatalf("viewMode = %d, want ViewKeys", updated.viewMode)
	}
	if updated.keysView == nil || !updated.keysView.hostRefs {
		t.Fatalf("keysView = %#v, want linked hosts state preserved", updated.keysView)
	}
}

func TestRootKeysHostEditEscReturnsToLinkedHosts(t *testing.T) {
	configFile := t.TempDir() + "/config"
	if err := os.WriteFile(configFile, []byte("Host dev.tisit.vm\n  HostName 127.0.0.1\n  User root\n"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	stubKeyInventoryForUITest(t, []keypkg.InventoryItem{
		{
			Path:        "/tmp/id_ed25519_demo",
			Fingerprint: "SHA256:test",
			Algorithm:   "ED25519",
			References: []keypkg.Reference{
				{Host: "dev.tisit.vm"},
			},
		},
	}, nil)

	m := newKeyWorkflowRootModel()
	m.configFile = configFile
	updated := openLoadedKeysView(t, m)

	updatedModel, _ := updated.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = updatedModel.(Model)
	updatedModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("h")})
	updated = updatedModel.(Model)
	updatedModel, cmd := updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want open host edit callback")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.viewMode != ViewEdit {
		t.Fatalf("viewMode = %d, want ViewEdit", updated.viewMode)
	}

	updatedModel, cmd = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = updatedModel.(Model)
	if cmd == nil {
		t.Fatalf("cmd = nil, want cancel callback")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.viewMode != ViewKeys {
		t.Fatalf("viewMode = %d, want ViewKeys", updated.viewMode)
	}
	if updated.keysView == nil || !updated.keysView.hostRefs {
		t.Fatalf("keysView = %#v, want linked hosts state preserved", updated.keysView)
	}
}

func TestAddFormCompactViewAtHeight15(t *testing.T) {
	form := NewAddForm("", NewStyles(80), 80, 15, "")
	view := form.View()
	if strings.Contains(view, "Terminal height is too small") {
		t.Fatalf("unexpected height warning in compact mode:\n%s", view)
	}
	if !strings.Contains(view, "Compact mode") {
		t.Fatalf("compact mode marker missing:\n%s", view)
	}
}

func TestEditFormCompactViewAtHeight15(t *testing.T) {
	form := newTestEditForm()
	form.height = 15
	view := form.View()
	if strings.Contains(view, "Terminal height is too small") {
		t.Fatalf("unexpected height warning in compact mode:\n%s", view)
	}
	if !strings.Contains(view, "Compact mode") {
		t.Fatalf("compact mode marker missing:\n%s", view)
	}
}

func TestKeyPickerViewUsesBasenameAndTildePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	picker := &keyPickerModel{
		items: []keypkg.InventoryItem{
			{
				Path:          home + "/.ssh/id_ed25519_demo",
				PublicKeyPath: home + "/.ssh/id_ed25519_demo.pub",
				Permissions:   "0600",
				Fingerprint:   "SHA256:testfingerprint",
				Algorithm:     "ED25519",
				References: []keypkg.Reference{
					{Host: "dev.tisit.vm"},
					{Host: "github.com"},
				},
			},
		},
		selected:  0,
		styles:    NewStyles(100),
		width:     100,
		height:    30,
		backLabel: "Edit Host",
		title:     "For dev.tisit.vm",
	}

	view := picker.View()
	if !strings.Contains(view, "Name") {
		t.Fatalf("name header missing:\n%s", view)
	}
	if strings.Contains(view, "Used By") {
		t.Fatalf("unexpected Used By label:\n%s", view)
	}
	if strings.Contains(view, "Selected:") {
		t.Fatalf("unexpected Selected label:\n%s", view)
	}
	if strings.Contains(view, "Showing explicit IdentityFile references only.") {
		t.Fatalf("unexpected disclaimer line:\n%s", view)
	}
	if !strings.Contains(view, "id_ed25519_demo") {
		t.Fatalf("basename missing:\n%s", view)
	}
	if !strings.Contains(view, "~/.ssh/id_ed25519_demo") {
		t.Fatalf("tilde path missing:\n%s", view)
	}
	if !strings.Contains(view, "2 hosts") {
		t.Fatalf("reference count summary missing:\n%s", view)
	}
	if !strings.Contains(view, "← back to Edit Host") {
		t.Fatalf("back label missing:\n%s", view)
	}
	if !strings.Contains(view, "esc/← back") {
		t.Fatalf("left-arrow back hint missing:\n%s", view)
	}
}

func TestInspectViewShowsPathActions(t *testing.T) {
	keysView := &keysViewModel{
		browser: &keyPickerModel{
			items: []keypkg.InventoryItem{
				{
					Path:        "/tmp/id_ed25519_demo",
					Fingerprint: "SHA256:test",
					Algorithm:   "ED25519",
				},
			},
			selected: 0,
			loading:  false,
		},
		styles:  NewStyles(100),
		width:   100,
		height:  30,
		inspect: true,
	}

	view := keysView.View()
	if !strings.Contains(view, "y copy path") {
		t.Fatalf("copy path hint missing:\n%s", view)
	}
	if !strings.Contains(view, "o reveal path") {
		t.Fatalf("reveal path hint missing:\n%s", view)
	}
	if !strings.Contains(view, "v deploy to host") {
		t.Fatalf("deploy hint missing:\n%s", view)
	}
}

func TestKeyBrowserViewOmitsChooseHint(t *testing.T) {
	browser := NewKeyBrowser("Main screen", NewStyles(100), 100, 30, "")
	browser.loading = false
	browser.items = []keypkg.InventoryItem{
		{Path: "/tmp/id_ed25519_demo", Fingerprint: "SHA256:test", Algorithm: "ED25519"},
	}

	view := browser.View()
	if strings.Contains(view, "enter choose") {
		t.Fatalf("unexpected chooser hint in browse mode:\n%s", view)
	}
	if !strings.Contains(view, "g generate") {
		t.Fatalf("generate hint missing:\n%s", view)
	}
	if !strings.Contains(view, "/tmp/id_ed25519_demo") {
		t.Fatalf("selected path missing in browse mode:\n%s", view)
	}
}

func TestKeyPickerViewKeepsFullPathOutsideHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	picker := &keyPickerModel{
		items: []keypkg.InventoryItem{
			{
				Path:        "/opt/keys/id_vm_custom",
				Permissions: "0600",
				Fingerprint: "SHA256:testfingerprint",
				Algorithm:   "ED25519",
			},
		},
		selected: 0,
		styles:   NewStyles(100),
		width:    100,
		height:   30,
	}

	view := picker.View()
	if !strings.Contains(view, "/opt/keys/id_vm_custom") {
		t.Fatalf("full path missing for non-home key:\n%s", view)
	}
}

func newTestEditForm() *editFormModel {
	inputs := make([]textinput.Model, 10)
	for i := range inputs {
		inputs[i] = textinput.New()
	}

	return &editFormModel{
		inputs:     inputs,
		focusArea:  focusAreaProperties,
		focused:    3,
		currentTab: 0,
		styles:     NewStyles(80),
		width:      80,
		height:     24,
	}
}

func openLoadedAddPicker(t *testing.T, form *addFormModel) *addFormModel {
	t.Helper()

	updated, cmd := form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if updated.keyPicker == nil {
		t.Fatalf("keyPicker = nil, want picker to open")
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want async load command")
	}

	msg := cmd()
	updated, _ = updated.Update(msg)
	if updated.keyPicker == nil || updated.keyPicker.loading {
		t.Fatalf("picker still loading after async load")
	}
	return updated
}

func openLoadedEditPicker(t *testing.T, form *editFormModel) *editFormModel {
	t.Helper()

	updatedModel, cmd := form.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated := updatedModel.(*editFormModel)
	if updated.keyPicker == nil {
		t.Fatalf("keyPicker = nil, want picker to open")
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want async load command")
	}

	msg := cmd()
	updatedModel, _ = updated.Update(msg)
	updated = updatedModel.(*editFormModel)
	if updated.keyPicker == nil || updated.keyPicker.loading {
		t.Fatalf("picker still loading after async load")
	}
	return updated
}

func openLoadedKeysView(t *testing.T, model Model) Model {
	t.Helper()

	updatedModel, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("K")})
	updated := updatedModel.(Model)
	if updated.keysView == nil {
		t.Fatalf("keysView = nil, want keys view to open")
	}
	if cmd == nil {
		t.Fatalf("cmd = nil, want async inventory load")
	}

	updatedModel, _ = updated.Update(cmd())
	updated = updatedModel.(Model)
	if updated.keysView == nil || updated.keysView.browser == nil || updated.keysView.browser.loading {
		t.Fatalf("keysView still loading after async load")
	}

	return updated
}

func newKeyWorkflowRootModel() Model {
	return Model{
		viewMode:    ViewList,
		ready:       true,
		width:       80,
		height:      24,
		styles:      NewStyles(80),
		searchInput: textinput.New(),
		table:       table.New(),
	}
}
