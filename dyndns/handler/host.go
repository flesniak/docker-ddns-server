package handler

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	l "github.com/labstack/gommon/log"

	"github.com/labstack/echo/v4"
	"github.com/w3K-one/docker-ddns-server/dyndns/model"
	"github.com/w3K-one/docker-ddns-server/dyndns/nswrapper"
	"gorm.io/gorm"
)

const (
	UNAUTHORIZED = "You are not allowed to view that content"
)

// GetHost fetches a host from the database by "id".
func (h *Handler) GetHost(c echo.Context) (err error) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	host := &model.Host{}
	if err = h.DB.First(host, id).Error; err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	// Display site
	return c.JSON(http.StatusOK, id)
}

// ListHosts fetches all hosts from database, performs an on-the-fly migration to lowercase, and lists them on the website.
func (h *Handler) ListHosts(c echo.Context) (err error) {
	var hosts []model.Host
	if err = h.DB.Find(&hosts).Error; err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	var changesMade []string
	needsMigration := false

	// Use a map to track existing lowercase hostnames to detect conflicts
	existingLowercaseHosts := make(map[string]bool)
	for _, host := range hosts {
		// Key for host map is a combination of hostname and domain
		hostKey := fmt.Sprintf("%s.%s", host.Hostname, host.Domain)
		existingLowercaseHosts[hostKey] = true
	}

	// Transaction to perform all updates at once for data integrity
	err = h.DB.Transaction(func(tx *gorm.DB) error {
		for i := range hosts {
			originalHostname := hosts[i].Hostname
			originalUsername := hosts[i].UserName

			lowerHostname := strings.ToLower(originalHostname)
			lowerUsername := strings.ToLower(originalUsername)

			isHostnameLower := originalHostname == lowerHostname
			isUsernameLower := originalUsername == lowerUsername

			if isHostnameLower && isUsernameLower {
				continue // Skip if already lowercase
			}

			needsMigration = true
			hostToUpdate := &hosts[i]

			// --- Handle Hostname Migration ---
			if !isHostnameLower {
				finalHostname := lowerHostname
				hostKey := fmt.Sprintf("%s.%s", finalHostname, hostToUpdate.Domain)
				if _, exists := existingLowercaseHosts[hostKey]; exists {
					for j := 1; ; j++ {
						newHostname := fmt.Sprintf("%s%d", lowerHostname, j)
						newHostKey := fmt.Sprintf("%s.%s", newHostname, hostToUpdate.Domain)
						if _, existsInner := existingLowercaseHosts[newHostKey]; !existsInner {
							finalHostname = newHostname
							break
						}
					}
				}
				hostToUpdate.Hostname = finalHostname
				// Add new name to map to prevent collisions within the same run
				existingLowercaseHosts[fmt.Sprintf("%s.%s", finalHostname, hostToUpdate.Domain)] = true
				changesMade = append(changesMade, fmt.Sprintf("Hostname '%s' was changed to '%s'.", originalHostname, finalHostname))
			}

			// --- Handle Username Migration ---
			if !isUsernameLower {
				hostToUpdate.UserName = lowerUsername // Simply convert to lowercase
				changesMade = append(changesMade, fmt.Sprintf("Username '%s' for host '%s' was changed to '%s'.", originalUsername, hostToUpdate.Hostname, lowerUsername))
			}

			if err := tx.Save(hostToUpdate).Error; err != nil {
				return err // Rollback on error
			}
		}
		return nil // Commit
	})

	if err != nil {
		return c.JSON(http.StatusInternalServerError, Error{Message: "Failed to migrate database entries: " + err.Error()})
	}

	// If a migration happened, re-query to show the final, updated list
	if needsMigration {
		if err = h.DB.Find(&hosts).Error; err != nil {
			return c.JSON(http.StatusBadRequest, &Error{err.Error()})
		}
	}

	migrationReport := ""
	if len(changesMade) > 0 {
		migrationReport = strings.Join(changesMade, "\n")
	}

	return c.Render(http.StatusOK, "listhosts", echo.Map{
		"hosts":           &hosts,
		"title":           h.Title,
		"logoPath":        h.LogoPath,
		"migrationReport": migrationReport,
		"poweredBy":       h.PoweredBy,
		"poweredByUrl":    h.PoweredByUrl,
	})
}

// AddHost just renders the "add host" website.
func (h *Handler) AddHost(c echo.Context) (err error) {
	// Get a list of existing hosts as possible tracking targets
	var hosts []model.Host
	if err = h.DB.Find(&hosts).Error; err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	otherHosts := make(map[string]uint)
	for _, h := range hosts {
		name := fmt.Sprintf("%s.%s", h.Hostname, h.Domain)
		otherHosts[name] = h.ID
	}

	return c.Render(http.StatusOK, "edithost", echo.Map{
		"addEdit":      "add",
		"config":       h.Config,
		"title":        h.Title,
		"logoPath":     h.LogoPath,
		"poweredBy":    h.PoweredBy,
		"poweredByUrl": h.PoweredByUrl,
		"otherHosts":   otherHosts,
	})
}

// EditHost fetches a host by "id" and renders the "edit host" website.
func (h *Handler) EditHost(c echo.Context) (err error) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	host := &model.Host{}
	if err = h.DB.First(host, id).Error; err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	// Get a list of existing hosts as possible tracking targets
	var hosts []model.Host
	if err = h.DB.Find(&hosts).Error; err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	otherHosts := make(map[string]uint)
	for _, h := range hosts {
		if h != *host {
			name := fmt.Sprintf("%s.%s", h.Hostname, h.Domain)
			otherHosts[name] = h.ID
		}
	}

	return c.Render(http.StatusOK, "edithost", echo.Map{
		"host":         host,
		"addEdit":      "edit",
		"config":       h.Config,
		"title":        h.Title,
		"logoPath":     h.LogoPath,
		"poweredBy":    h.PoweredBy,
		"poweredByUrl": h.PoweredByUrl,
		"otherHosts":   otherHosts,
	})
}

// CreateHost validates the host data from the "add host" website,
// adds the host entry to the database,
// and adds the entry to the DNS server.
func (h *Handler) CreateHost(c echo.Context) (err error) {
	host := &model.Host{}
	if err = c.Bind(host); err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	// Enforce lowercase for new entries
	host.Hostname = strings.ToLower(host.Hostname)
	host.UserName = strings.ToLower(host.UserName)

	// TrackingHostID is not bound to host, maybe because it's uint -> do it manually
	trackingHostID := c.FormValue("trackinghostid")
	if trackingHostID != "none" {
		id, err := strconv.Atoi(trackingHostID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, &Error{err.Error()})
		}

		trackingHost := &model.Host{}
		if err := h.DB.First(trackingHost, id).Error; err != nil {
			return c.JSON(http.StatusBadRequest, &Error{err.Error()})
		}
		// h.DB.Model(host).Association("TrackingHost").Replace(trackingHost)
		host.TrackingHostID = uint(id)
	} else {
		// h.DB.Model(host).Association("TrackingHost").Clear()
		host.TrackingHostID = 0
	}

	if err = c.Validate(host); err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	if err = h.checkUniqueHostname(host.Hostname, host.Domain); err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}
	host.LastUpdate = time.Now()
	if err = h.DB.Create(host).Error; err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	// If a ip is set create dns entry
	if host.Ip4 != "" {
		ipType := nswrapper.GetIPType(host.Ip4)
		if ipType != "A" {
			return c.JSON(http.StatusBadRequest, &Error{fmt.Sprintf("ip %s is not a valid ipv4 address", host.Ip4)})
		}

		if err = nswrapper.UpdateRecord(host.Hostname, host.Ip4, ipType, host.Domain, host.Ttl, h.AllowWildcard); err != nil {
			return c.JSON(http.StatusBadRequest, &Error{err.Error()})
		}
	}

	if host.Ip6 != "" {
		ipType := nswrapper.GetIPType(host.Ip6)
		if ipType != "AAAA" {
			return c.JSON(http.StatusBadRequest, &Error{fmt.Sprintf("ip %s is not a valid ipv6 address", host.Ip6)})
		}

		if err = nswrapper.UpdateRecord(host.Hostname, host.Ip6, ipType, host.Domain, host.Ttl, h.AllowWildcard); err != nil {
			return c.JSON(http.StatusBadRequest, &Error{err.Error()})
		}
	}

	return c.JSON(http.StatusOK, host)
}

// UpdateHost validates the host data from the "edit host" website,
// and compares the host data with the entry in the database by "id".
// If anything has changed the database and DNS entries for the host will be updated.
func (h *Handler) UpdateHost(c echo.Context) (err error) {
	hostUpdate := &model.Host{}
	if err = c.Bind(hostUpdate); err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	// Enforce lowercase for updated entries
	hostUpdate.Hostname = strings.ToLower(hostUpdate.Hostname)
	hostUpdate.UserName = strings.ToLower(hostUpdate.UserName)

	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	host := &model.Host{}
	if err = h.DB.First(host, id).Error; err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	// TrackingHostID is not bound to hostUpdate, maybe because it's uint -> do it manually
	trackingHostID := c.FormValue("trackinghostid")
	if trackingHostID != "none" {
		id, err := strconv.Atoi(trackingHostID)
		if err != nil {
			return c.JSON(http.StatusBadRequest, &Error{err.Error()})
		}

		trackingHost := &model.Host{}
		if err := h.DB.First(trackingHost, id).Error; err != nil {
			return c.JSON(http.StatusBadRequest, &Error{err.Error()})
		}
		// h.DB.Model(hostUpdate).Association("TrackingHost").Replace(trackingHost)
		hostUpdate.TrackingHostID = uint(id)
	} else {
		// h.DB.Model(hostUpdate).Association("TrackingHost").Clear()
		hostUpdate.TrackingHostID = 0
	}

	forceRecordUpdate := host.UpdateHost(hostUpdate)
	if err = c.Validate(host); err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	if err = h.DB.Save(host).Error; err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	// If ip or ttl changed update dns entry
	if forceRecordUpdate {
		if host.Ip4 != "" {
			ipType := nswrapper.GetIPType(host.Ip4)
			if ipType != "A" {
				return c.JSON(http.StatusBadRequest, &Error{fmt.Sprintf("ip %s is not a valid ipv4 address", host.Ip4)})
			}

			if err = nswrapper.UpdateRecord(host.Hostname, host.Ip4, ipType, host.Domain, host.Ttl, h.AllowWildcard); err != nil {
				return c.JSON(http.StatusBadRequest, &Error{err.Error()})
			}
		}

		if host.Ip6 != "" {
			ipType := nswrapper.GetIPType(host.Ip6)
			if ipType != "AAAA" {
				return c.JSON(http.StatusBadRequest, &Error{fmt.Sprintf("ip %s is not a valid ipv6 address", host.Ip6)})
			}

			if err = nswrapper.UpdateRecord(host.Hostname, host.Ip6, ipType, host.Domain, host.Ttl, h.AllowWildcard); err != nil {
				return c.JSON(http.StatusBadRequest, &Error{err.Error()})
			}
		}
	}

	return c.JSON(http.StatusOK, host)
}

// DeleteHost fetches a host entry from the database by "id"
// and deletes the database and DNS server entry to it.
func (h *Handler) DeleteHost(c echo.Context) (err error) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	host := &model.Host{}
	if err = h.DB.First(host, id).Error; err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	err = h.DB.Transaction(func(tx *gorm.DB) error {
		if err = tx.Unscoped().Delete(host).Error; err != nil {
			return c.JSON(http.StatusBadRequest, &Error{err.Error()})
		}

		if err = tx.Where(&model.Log{HostID: uint(id)}).Delete(&model.Log{}).Error; err != nil {
			return c.JSON(http.StatusBadRequest, &Error{err.Error()})
		}

		if err = tx.Where(&model.CName{TargetID: uint(id)}).Delete(&model.CName{}).Error; err != nil {
			return c.JSON(http.StatusBadRequest, &Error{err.Error()})
		}

		return nil
	})
	if err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	if err = nswrapper.DeleteRecord(host.Hostname, host.Domain, h.AllowWildcard); err != nil {
		return c.JSON(http.StatusBadRequest, &Error{err.Error()})
	}

	return c.JSON(http.StatusOK, id)
}

// UpdateIP implements the update method called by the routers.
// Hostname, IP and senders IP are validated, a log entry is created
// and finally if everything is ok, the DNS Server will be updated
func (h *Handler) UpdateIP(c echo.Context) (err error) {
	host, ok := c.Get("updateHost").(*model.Host)
	if !ok {
		return c.String(http.StatusBadRequest, "badauth\n")
	}

	log := &model.Log{Status: false, Host: *host, TimeStamp: time.Now(), UserAgent: nswrapper.ShrinkUserAgent(c.Request().UserAgent())}
	log.SentIPs = c.QueryParam(("myip"))

	// Get caller IP
	log.CallerIP, _ = nswrapper.GetCallerIP(c.Request())
	if log.CallerIP == "" {
		log.CallerIP, _, err = net.SplitHostPort(c.Request().RemoteAddr)
		if err != nil {
			log.Message = "Bad Request: Unable to get caller IP"
			if err = h.CreateLogEntry(log); err != nil {
				l.Error(err)
			}

			return c.String(http.StatusBadRequest, "badrequest\n")
		}
	}

	// Validate hostname (already lowercased during authentication)
	hostname := strings.ToLower(c.QueryParam("hostname"))
	if hostname == "" || hostname != host.Hostname+"."+host.Domain {
		log.Message = "Hostname or combination of authenticated user and hostname is invalid"
		if err = h.CreateLogEntry(log); err != nil {
			l.Error(err)
		}

		return c.String(http.StatusBadRequest, "notfqdn\n")
	}

	// multiple IP addresses can be given and must use comma as separator
	sentIPs := strings.Split(log.SentIPs, ",")
	ipv4 := ""
	ipv6 := ""
	for _, sentIP := range sentIPs {
		// Get IP type
		ipType := nswrapper.GetIPType(sentIP)
		switch ipType {
		case "A":
			ipv4 = sentIP
		case "AAAA":
			ipv6 = sentIP
		default:
			log.Message = "Failed to parse sent ip"
			if err = h.CreateLogEntry(log); err != nil {
				l.Error(err)
			}
			continue // ignore unmatched IPs
		}
	}

	if ipv4 == "" && ipv6 == "" {
		log.Message = "Bad Request: No valid IP (neither v4 nor v6) found"
		if err = h.CreateLogEntry(log); err != nil {
			l.Error(err)
		}
		return c.String(http.StatusBadRequest, "badrequest\n")
	}

	// Add/update DNS records
	if ipv4 != "" {
		if err = nswrapper.UpdateRecord(log.Host.Hostname, ipv4, "A", log.Host.Domain, log.Host.Ttl, h.AllowWildcard); err != nil {
			log.Message = fmt.Sprintf("DNS error: %v", err)
			l.Error(log.Message)
			if err = h.CreateLogEntry(log); err != nil {
				l.Error(err)
			}
			return c.String(http.StatusBadRequest, "dnserr\n")
		}
		log.Host.Ip4 = ipv4
	}

	if ipv6 != "" {
		if err = nswrapper.UpdateRecord(log.Host.Hostname, ipv6, "AAAA", log.Host.Domain, log.Host.Ttl, h.AllowWildcard); err != nil {
			log.Message = fmt.Sprintf("DNS error: %v", err)
			l.Error(log.Message)
			if err = h.CreateLogEntry(log); err != nil {
				l.Error(err)
			}
			return c.String(http.StatusBadRequest, "dnserr\n")
		}
		log.Host.Ip6 = ipv6
	}

	// Update DB host entry
	log.Host.LastUpdate = log.TimeStamp

	if err = h.DB.Save(log.Host).Error; err != nil {
		return c.JSON(http.StatusBadRequest, "badrequest\n")
	}

	log.Status = true
	log.Message = "No errors occurred"
	if err = h.CreateLogEntry(log); err != nil {
		l.Error(err)
	}

	return c.String(http.StatusOK, "good\n")
}

func (h *Handler) checkUniqueHostname(hostname, domain string) error {
	hosts := new([]model.Host)
	if err := h.DB.Where(&model.Host{Hostname: hostname, Domain: domain}).Find(hosts).Error; err != nil {
		return err
	}

	if len(*hosts) > 0 {
		return fmt.Errorf("hostname already exists")
	}

	cnames := new([]model.CName)
	if err := h.DB.Preload("Target").Where(&model.CName{Hostname: hostname}).Find(cnames).Error; err != nil {
		return err
	}

	for _, cname := range *cnames {
		if cname.Target.Domain == domain {
			return fmt.Errorf("hostname already exists")
		}
	}

	return nil
}
