package model

import (
	"time"

	"gorm.io/gorm"
)

// Host is a dns host entry.
type Host struct {
	gorm.Model
	Hostname       string    `gorm:"unique_index:idx_host_domain;not null" form:"hostname" validate:"required,min=1"` // Allow 1 character hostnames
	Domain         string    `gorm:"unique_index:idx_host_domain;not null" form:"domain" validate:"required,fqdn"`
	Ip4            string    `form:"ip4" validate:"omitempty,ipv4"`
	Ip6            string    `form:"ip6" validate:"omitempty,ipv6"`
	Ttl            int       `form:"ttl" validate:"required,min=20,max=86400"`
	LastUpdate     time.Time `form:"lastupdate"`
	UserName       string    `gorm:"not null" form:"username" validate:"min=1"` // Allow 1 character usernames
	Password       string    `form:"password" validate:"min=6"`                 // Minimum 6 character passwords
	TrackingHostID uint
	TrackingHost   *Host  `gorm:"foreignkey:TrackingHostID" form:"trackinghost" validate:"omitempty,fqdn"`
	Ip6HostPart    string `form:"ip6hostpart" validate:"omitempty,ipv6"`
	Ip6HostSize    int    `gorm:"default=64" form:"ip6hostsize" validate:"min=1,max=127"`
}

// UpdateHost updates all fields of a host entry
// and sets a new LastUpdate date.
func (h *Host) UpdateHost(updateHost *Host) (updateRecord bool) {
	updateRecord = false

	// check all parameters that should result in a DNS record update if changed
	if h.Ip4 != updateHost.Ip4 || h.Ip6 != updateHost.Ip6 || h.Ttl != updateHost.Ttl || h.TrackingHostID != updateHost.TrackingHostID {
		updateRecord = true
		h.LastUpdate = time.Now()
	}

	h.Ip4 = updateHost.Ip4
	h.Ip6 = updateHost.Ip6
	h.Ttl = updateHost.Ttl
	h.UserName = updateHost.UserName
	h.Password = updateHost.Password
	h.TrackingHostID = updateHost.TrackingHostID

	return
}
