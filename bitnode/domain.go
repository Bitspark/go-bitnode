package bitnode

import (
	"fmt"
	"github.com/Bitspark/go-bitnode/util"
	"gopkg.in/yaml.v3"
	"log"
	"os"
	"path"
	"strings"
)

const DomSep = "."

type Compilable interface {
	// Compile transforms a compilable item into an internal structure.
	// domName is the name of the domain the current compilable is in, used to resolve references.
	// resolve specifies whether the compilable should be expanded (i.e., references or extends)
	Compile(dom *Domain, domName string, resolve bool) error

	// Reset marks the Compilable as dirty so that it can be recompiled.
	Reset()

	FullDomain() string
}

type Savable interface {
	Save(dom *Domain) error
}

type compilable struct {
	// compiled indicates whether it has been compiled, i.e. its reference has been resolved.
	compiled bool

	// compiling indicates it is currently being compiled to avoid a stack overflow.
	compiling bool
}

// PermissionGroup determines whether a user is part of this group.
type PermissionGroup struct {
	Public bool   `json:"public"`
	Auth   bool   `json:"auth"`
	Users  IDList `json:"users"`
	Groups IDList `json:"groups"`
}

// Contains determines if a user is part of the PermissionGroup.
func (pg *PermissionGroup) Contains(user ID, groups []ID) bool {
	if pg.Public {
		return true
	}
	if !user.IsNull() && pg.Auth {
		return true
	}
	if pg.Users != nil {
		if ok := util.Contains(pg.Users, user); ok {
			return true
		}
	}
	if pg.Groups != nil {
		for _, group := range groups {
			if ok := util.Contains(pg.Users, group); ok {
				return true
			}
		}
	}
	return false
}

// Permissions contains admin, extend and view permissions for specific users and groups.
type Permissions struct {
	// Owner owns this object and is always an Admin.
	Owner ID `json:"owner" yaml:"owner"`

	// Admin can do anything.
	Admin PermissionGroup `json:"admin" yaml:"admin"`

	// Extend can change the contents in a way that does not remove any data.
	Extend PermissionGroup `json:"extend" yaml:"extend"`

	// View can view the contents.
	View PermissionGroup `json:"view" yaml:"view"`
}

func (p *Permissions) HavePermissions(permission string, creds Credentials) bool {
	admin := creds.Admin
	if admin {
		return true
	}
	user := creds.User.ID
	groups := creds.Groups
	if p == nil {
		return false
	}
	if !p.Owner.IsNull() && p.Owner == user {
		return true
	}
	if permission == "owner" {
		return false
	}
	if !p.Owner.IsNull() && p.Owner == user {
		return true
	}
	if p.Admin.Contains(user, groups) {
		return true
	}
	switch permission {
	case "extend":
		return p.Extend.Contains(user, groups)
	case "view":
		return p.View.Contains(user, groups)
	}
	return false
}

// Domain contains definitions and hooks to the native environment. It is independent of states.
type Domain struct {
	compilable `json:"-" yaml:"-"`

	// FullName of the domain.
	FullName string `json:"fullName" yaml:"-"`

	// Parent domain.
	Parent *Domain `json:"-" yaml:"-"`

	// Domains are child domains of this domain.
	Domains []*Domain `json:"-" yaml:"-"`

	// Name of the domain.
	Name string `json:"name" yaml:"name"`

	// Description of the domain.
	Description string `json:"description" yaml:"description"`

	// Permissions of this domain.
	Permissions *Permissions `json:"permissions" yaml:"permissions"`

	// Types inside this domain.
	Types []*Type `json:"types" yaml:"types"`

	// Interfaces inside this domain.
	Interfaces []*Interface `json:"interfaces" yaml:"interfaces"`

	// Blueprints inside this domain.
	Sparkables []*Sparkable `json:"blueprints" yaml:"blueprints"`

	FilePath string `json:"-" yaml:"-"`
}

func NewDomain() *Domain {
	return &Domain{
		FullName: "",
		Permissions: &Permissions{
			Owner:  ID{},
			Admin:  PermissionGroup{},
			Extend: PermissionGroup{},
			View:   PermissionGroup{},
		},
	}
}

func (dom *Domain) LoadFromDir(dir string, recursive bool) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	hasYAML := false

	for _, f := range files {
		chPath := path.Join(dir, f.Name())
		if f.IsDir() {
			if !recursive {
				continue
			}
			if strings.HasSuffix(f.Name(), "_test") {
				log.Printf("Skip directory: %s", chPath)
				continue
			}
			childDom := &Domain{
				Parent: dom,
			}
			childDom.Name = f.Name()
			if dom.FullName != "" {
				childDom.FullName = dom.FullName + DomSep + f.Name()
			} else {
				childDom.FullName = f.Name()
			}
			if err := childDom.LoadFromDir(chPath, true); err != nil {
				log.Printf("Loading domain %s from directory %s: %v", childDom.FullName, chPath, err)
				continue
			}
			dom.Domains = append(dom.Domains, childDom)
		} else {
			if !strings.HasSuffix(f.Name(), ".yml") && !strings.HasSuffix(f.Name(), ".yaml") {
				continue
			}
			if err := dom.LoadFromFile(chPath); err != nil {
				return err
			}
			hasYAML = true
		}
	}

	if !hasYAML {
		return fmt.Errorf("no definitions found")
	}

	return nil
}

func (dom *Domain) LoadFromFile(file string) error {
	if dom.FilePath != "" {
		return fmt.Errorf("already have path when loading %s: %s", file, dom.FilePath)
	}

	chDefsBytes, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("reading definitions from %s: %v", file, err)
	}

	defs := Domain{}
	if err := yaml.Unmarshal(chDefsBytes, &defs); err != nil {
		yamlFactories = nil
		return fmt.Errorf("parsing definitions from %s: %v", file, err)
	}

	dom.Permissions = defs.Permissions

	if err := dom.addDefinitions(defs); err != nil {
		return err
	}

	dom.FilePath = file

	return nil
}

func (dom *Domain) Compile() error {
	if dom.Parent != nil && dom.FullName == "" {
		dom.FullName = dom.Parent.FullName + DomSep + dom.Name
	}
	for _, t := range dom.Types {
		if err := t.Compile(dom, dom.FullName, true); err != nil {
			return fmt.Errorf("compile type %s in domain %s: %v", t.Name, dom.FullName, err)
		}
	}
	for _, i := range dom.Interfaces {
		if err := i.Compile(dom, dom.FullName, true); err != nil {
			return fmt.Errorf("compile interface %s in domain %s: %v", i.Name, dom.FullName, err)
		}
	}
	for _, m := range dom.Sparkables {
		if err := m.Compile(dom, dom.FullName, true); err != nil {
			return fmt.Errorf("compile impl %s in domain %s: %v", m.Name, dom.FullName, err)
		}
	}
	for _, d := range dom.Domains {
		if err := d.Compile(); err != nil {
			return err
		}
	}
	dom.compiled = true
	return nil
}

func (dom *Domain) Copy() *Domain {
	return dom.copy(nil)
}

func (dom *Domain) Root() *Domain {
	if dom == nil {
		return nil
	}
	if dom.Parent == nil {
		return dom
	}
	return dom.Parent.Root()
}

func (dom *Domain) Save() error {
	if yamlBts, err := yaml.Marshal(*dom); err != nil {
		return fmt.Errorf("parsing definitions from %s: %v", dom.FilePath, err)
	} else {
		if err := os.MkdirAll(path.Dir(dom.FilePath), os.ModePerm); err != nil {
			return err
		}
		if err := os.WriteFile(dom.FilePath, yamlBts, os.ModePerm); err != nil {
			return err
		}
	}

	return nil
}

func (dom *Domain) Delete() error {
	if dom.FilePath == "" {
		return nil
	}

	return os.RemoveAll(path.Dir(dom.FilePath))
}

func (dom *Domain) SaveAll() error {
	if err := dom.Save(); err != nil {
		return err
	}

	for _, c := range dom.Domains {
		if err := c.SaveAll(); err != nil {
			return err
		}
	}

	return nil
}

func (dom *Domain) AddFullDomain(domPath string, fullName string) error {
	if domPath == "" {
		domPath = path.Dir(dom.FilePath)
	}

	doms := strings.Split(fullName, DomSep)
	d := doms[0]

	nd, err := dom.GetDomain(dom.FullName + DomSep + d)
	if err != nil {
		nd, err = dom.AddDomain(d)
		if err != nil {
			return err
		} else {
			nd.FilePath = path.Join(domPath, fmt.Sprintf("%s.yml", d))
		}
	}

	if len(doms) <= 1 {
		return nil
	}

	return nd.AddFullDomain(path.Join(domPath, doms[1]), strings.Join(doms[1:], DomSep))
}

func (dom *Domain) AddDomain(name string) (*Domain, error) {
	return dom.addDomain(name)
}

func (dom *Domain) GetDomain(name string) (*Domain, error) {
	return dom.getDomain(strings.Split(name, DomSep))
}

func (dom *Domain) CreateDomain(name string, perms Permissions) error {
	frags := strings.Split(name, DomSep)
	d, err := dom.getDomain(frags[:len(frags)-1])
	if err != nil {
		return err
	}
	return d.createDomain(frags[len(frags)-1], perms)
}

func (dom *Domain) DeleteDomain(name string) error {
	frags := strings.Split(name, DomSep)
	d, err := dom.getDomain(frags[:len(frags)-1])
	if err != nil {
		return err
	}
	return d.deleteDomain(frags[len(frags)-1])
}

func (dom *Domain) MustGetType(name string) *Type {
	d, err := dom.GetType(name)
	if err != nil {
		panic(err)
	}
	return d
}

func (dom *Domain) GetType(name string) (*Type, error) {
	frags := strings.Split(name, DomSep)
	d, err := dom.getDomain(frags[:len(frags)-1])
	if err != nil {
		return nil, err
	}
	return d.getType(frags[len(frags)-1])
}

func (dom *Domain) CreateType(name string, perms Permissions) (*Type, error) {
	frags := strings.Split(name, DomSep)
	d, err := dom.getDomain(frags[:len(frags)-1])
	if err != nil {
		return nil, err
	}
	return d.createType(frags[len(frags)-1], perms)
}

func (dom *Domain) DeleteType(name string) error {
	frags := strings.Split(name, DomSep)
	d, err := dom.getDomain(frags[:len(frags)-1])
	if err != nil {
		return err
	}
	return d.deleteType(frags[len(frags)-1])
}

func (dom *Domain) GetInterface(name string) (*Interface, error) {
	frags := strings.Split(name, DomSep)
	d, err := dom.getDomain(frags[:len(frags)-1])
	if err != nil {
		return nil, err
	}
	return d.getInterface(frags[len(frags)-1])
}

func (dom *Domain) CreateInterface(name string, perms Permissions) (*Interface, error) {
	frags := strings.Split(name, DomSep)
	d, err := dom.getDomain(frags[:len(frags)-1])
	if err != nil {
		return nil, err
	}
	return d.createInterface(frags[len(frags)-1], perms)
}

func (dom *Domain) DeleteInterface(name string) error {
	frags := strings.Split(name, DomSep)
	d, err := dom.getDomain(frags[:len(frags)-1])
	if err != nil {
		return err
	}
	return d.deleteInterface(frags[len(frags)-1])
}

func (dom *Domain) GetSparkable(name string) (*Sparkable, error) {
	frags := strings.Split(name, DomSep)
	d, err := dom.getDomain(frags[:len(frags)-1])
	if err != nil {
		return nil, err
	}
	return d.getSparkable(frags[len(frags)-1])
}

func (dom *Domain) CreateSparkable(name string, perms Permissions) (*Sparkable, error) {
	frags := strings.Split(name, DomSep)
	d, err := dom.getDomain(frags[:len(frags)-1])
	if err != nil {
		return nil, err
	}
	return d.createSparkable(frags[len(frags)-1], perms)
}

func (dom *Domain) DeleteSparkable(name string) error {
	frags := strings.Split(name, DomSep)
	d, err := dom.getDomain(frags[:len(frags)-1])
	if err != nil {
		return err
	}
	return d.deleteSparkable(frags[len(frags)-1])
}

// Private

func (dom *Domain) addDefinitions(defs Domain) error {
	if defs.Name == "" {
		return fmt.Errorf("require domain contents name")
	}
	if dom.FullName == "" {
		dom.FullName = defs.Name
		dom.Name = defs.Name
	} else if dom.Name != defs.Name {
		return fmt.Errorf("contents for domain %s cannot be added to domain %s", defs.Name, dom.Name)
	}
	if dom.Description == "" {
		dom.Description = defs.Description
	}
	for _, d := range defs.Types {
		if err := dom.addType(d); err != nil {
			return err
		}
	}
	for _, d := range defs.Interfaces {
		if err := dom.addInterface(d); err != nil {
			return err
		}
	}
	for _, d := range defs.Sparkables {
		if err := dom.addSparkable(d); err != nil {
			return err
		}
	}
	return nil
}

func (dom *Domain) copy(parent *Domain) *Domain {
	libCpy := &Domain{
		FullName: dom.FullName,
		Parent:   parent,
	}
	*libCpy = *dom
	for c, l := range dom.Domains {
		l2 := l.copy(dom)
		dom.Domains[c] = l2
	}
	return libCpy
}

func (dom *Domain) getDomain(frags []string) (*Domain, error) {
	if len(frags) == 0 {
		return dom, nil
	}
	name := frags[0]
	if name == "" {
		return dom.getSubDomain(frags[1:])
	}
	return dom.Root().getSubDomain(frags)
}

func (dom *Domain) getSubDomain(frags []string) (*Domain, error) {
	if dom == nil {
		return nil, fmt.Errorf("domain %s not found inside nil domain", frags[0])
	}
	if len(frags) == 0 {
		return dom, nil
	}
	name := frags[0]
	if name == "" {
		return dom.getSubDomain(frags[1:])
	}
	if name == "$" {
		return dom.Parent.getSubDomain(frags[1:])
	}
	for _, d := range dom.Domains {
		if d.Name == name {
			return d.getSubDomain(frags[1:])
		}
	}
	return nil, fmt.Errorf("domain %s not found inside domain %s", frags[0], dom.FullName)
}

func (dom *Domain) addDomain(name string) (*Domain, error) {
	if err := checkName(name, 1, 12, false); err != nil {
		return nil, err
	}
	if _, err := dom.GetDomain(name); err == nil {
		return nil, fmt.Errorf("already have a sub-domain %s", name)
	}
	d := NewDomain()
	if dom.FullName != "" {
		d.FullName = dom.FullName + DomSep + name
	} else {
		d.FullName = name
	}
	d.Name = name
	d.Parent = dom
	dom.Domains = append(dom.Domains, d)
	return d, nil
}

func (dom *Domain) deleteDomain(name string) error {
	newDomains := []*Domain{}
	for _, sd := range dom.Domains {
		if sd.Name == name {
			if err := sd.Delete(); err != nil {
				return err
			}
			continue
		}
		newDomains = append(newDomains, sd)
	}
	dom.Domains = newDomains
	return nil
}

func (dom *Domain) addType(domType *Type) error {
	domType.Domain = dom.FullName
	dom.Types = append(dom.Types, domType)
	if domType.Permissions == nil {
		domType.Permissions = &Permissions{
			Owner:  ID{},
			Admin:  PermissionGroup{},
			Extend: PermissionGroup{},
			View:   PermissionGroup{},
		}
	}
	return nil
}

func (dom *Domain) getType(name string) (*Type, error) {
	for _, d := range dom.Types {
		if d.Name == name {
			return d, nil
		}
	}
	return nil, fmt.Errorf("type not found in domain %s: %s", dom.FullName, name)
}

func (dom *Domain) addInterface(domInterface *Interface) error {
	domInterface.Domain = dom.FullName
	dom.Interfaces = append(dom.Interfaces, domInterface)
	if domInterface.Permissions == nil {
		domInterface.Permissions = &Permissions{
			Owner:  ID{},
			Admin:  PermissionGroup{},
			Extend: PermissionGroup{},
			View:   PermissionGroup{},
		}
	}
	return nil
}

func (dom *Domain) getInterface(name string) (*Interface, error) {
	for _, d := range dom.Interfaces {
		if d.Name == name {
			return d, nil
		}
	}
	return nil, fmt.Errorf("interface %s not found in domain %s", name, dom.FullName)
}

func (dom *Domain) addSparkable(domSparkable *Sparkable) error {
	domSparkable.Domain = dom.FullName
	dom.Sparkables = append(dom.Sparkables, domSparkable)
	if domSparkable.Permissions == nil {
		domSparkable.Permissions = &Permissions{
			Owner:  ID{},
			Admin:  PermissionGroup{},
			Extend: PermissionGroup{},
			View:   PermissionGroup{},
		}
	}
	return nil
}

func (dom *Domain) getSparkable(name string) (*Sparkable, error) {
	for _, d := range dom.Sparkables {
		if d.Name == name {
			return d, nil
		}
	}
	return nil, fmt.Errorf("sparkable not found in domain %s: %s", dom.FullName, name)
}

func (dom *Domain) deleteSparkable(name string) error {
	newSparkables := []*Sparkable{}
	found := false
	for _, d := range dom.Sparkables {
		if d.Name == name {
			found = true
			continue
		}
		newSparkables = append(newSparkables, d)
	}
	if !found {
		return fmt.Errorf("sparkable not found in domain %s: %s", dom.FullName, name)
	}
	dom.Sparkables = newSparkables
	_ = dom.Save()
	return nil
}

func (dom *Domain) createSparkable(name string, perms Permissions) (*Sparkable, error) {
	if err := checkName(name, 1, 24, true); err != nil {
		return nil, err
	}
	newSparkables := []*Sparkable{}
	for _, d := range dom.Sparkables {
		if d.Name == name {
			return nil, fmt.Errorf("this sparkable already exists")
		}
		newSparkables = append(newSparkables, d)
	}
	sparkable := &Sparkable{
		RawSparkable: RawSparkable{
			Name:        name,
			Domain:      dom.FullName,
			Permissions: &perms,
		},
	}
	if err := sparkable.Compile(dom, dom.FullName, true); err != nil {
		return nil, err
	}
	newSparkables = append(newSparkables, sparkable)
	dom.Sparkables = newSparkables
	_ = dom.Save()
	return sparkable, nil
}

func (dom *Domain) createDomain(name string, perms Permissions) error {
	if err := checkName(name, 1, 12, false); err != nil {
		return err
	}
	newDomains := []*Domain{}
	for _, d := range dom.Domains {
		if d.Name == name {
			return fmt.Errorf("this domain already exists")
		}
		newDomains = append(newDomains, d)
	}
	domain := &Domain{
		Parent:      dom,
		Name:        name,
		Permissions: &perms,
	}
	if dom.FilePath != "" {
		baseDir := path.Join(path.Dir(dom.FilePath), name)
		if err := os.MkdirAll(baseDir, os.ModePerm); err != nil {
			return err
		}
		domain.FilePath = path.Join(baseDir, name+".yml")
	}
	if err := domain.Compile(); err != nil {
		return err
	}
	newDomains = append(newDomains, domain)
	dom.Domains = newDomains
	_ = dom.Save()
	_ = domain.Save()
	return nil
}

func (dom *Domain) deleteInterface(name string) error {
	newInterfaces := []*Interface{}
	found := false
	for _, d := range dom.Interfaces {
		if d.Name == name {
			found = true
			continue
		}
		newInterfaces = append(newInterfaces, d)
	}
	if !found {
		return fmt.Errorf("interface not found in domain %s: %s", dom.FullName, name)
	}
	dom.Interfaces = newInterfaces
	_ = dom.Save()
	return nil
}

func (dom *Domain) createInterface(name string, perms Permissions) (*Interface, error) {
	if err := checkName(name, 1, 24, true); err != nil {
		return nil, err
	}
	newInterfaces := []*Interface{}
	for _, d := range dom.Interfaces {
		if d.Name == name {
			return nil, fmt.Errorf("this interface already exists")
		}
		newInterfaces = append(newInterfaces, d)
	}
	interf := &Interface{
		RawInterface: RawInterface{
			Name:        name,
			Domain:      dom.FullName,
			Permissions: &perms,
		},
	}
	if err := interf.Compile(dom, dom.FullName, true); err != nil {
		return nil, err
	}
	newInterfaces = append(newInterfaces, interf)
	dom.Interfaces = newInterfaces
	_ = dom.Save()
	return interf, nil
}

func (dom *Domain) deleteType(name string) error {
	newTypes := []*Type{}
	found := false
	for _, d := range dom.Types {
		if d.Name == name {
			found = true
			continue
		}
		newTypes = append(newTypes, d)
	}
	if !found {
		return fmt.Errorf("type not found in domain %s: %s", dom.FullName, name)
	}
	dom.Types = newTypes
	_ = dom.Save()
	return nil
}

func (dom *Domain) createType(name string, perms Permissions) (*Type, error) {
	if err := checkName(name, 1, 24, false); err != nil {
		return nil, err
	}
	newTypes := []*Type{}
	for _, d := range dom.Types {
		if d.Name == name {
			return nil, fmt.Errorf("this type already exists")
		}
		newTypes = append(newTypes, d)
	}
	tp := &Type{
		RawType: RawType{
			Name:        name,
			Domain:      dom.FullName,
			Permissions: &perms,
		},
	}
	if err := tp.Compile(dom, dom.FullName, true); err != nil {
		return nil, err
	}
	newTypes = append(newTypes, tp)
	dom.Types = newTypes
	_ = dom.Save()
	return tp, nil
}

func checkName(name string, minLength int, maxLength int, capital bool) error {
	if len(name) < minLength {
		return fmt.Errorf("name must be at longer than %d characters", minLength)
	}
	if len(name) > maxLength {
		return fmt.Errorf("name must be be shorter than %d characters", maxLength)
	}
	if capital {
		if name[0] < 'A' || name[0] > 'Z' {
			return fmt.Errorf("name must start with an upper case character (A-Z)")
		}
	} else {
		if name[0] < 'a' || name[0] > 'z' {
			return fmt.Errorf("name must start with a lower case character (a-z)")
		}
	}
	if !util.IsAlphanumeric(name) {
		return fmt.Errorf("name must not contain special characters")
	}
	return nil
}
