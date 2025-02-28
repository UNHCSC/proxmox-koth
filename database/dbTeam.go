package database

import (
	"encoding/json"
	"errors"
)

var ErrTeamExists = errors.New("team already exists")
var ErrTeamNotFound = errors.New("team not found")

const TEAMS_STATEMENT = `CREATE TABLE IF NOT EXISTS teams (
	name TEXT PRIMARY KEY NOT NULL,
	container_ip TEXT NOT NULL,
	container_id INTEGER NOT NULL,
	score INTEGER NOT NULL,
	uptimeChecksTotal INTEGER DEFAULT 0,
	uptimeChecksPassed INTEGER DEFAULT 0,
	serviceChecksTotal INTEGER DEFAULT 0,
	serviceChecksPassed INTEGER DEFAULT 0
);`

const INSERT_TEAM_STATEMENT = `INSERT INTO teams (name, container_ip, container_id, score) VALUES (?, ?, ?, ?);`
const SELECT_TEAM_STATEMENT = `SELECT name, container_ip, container_id, score, uptimeChecksTotal, uptimeChecksPassed, serviceChecksTotal, serviceChecksPassed FROM teams WHERE name = ?;`
const DELETE_TEAM_STATEMENT = `DELETE FROM teams WHERE name = ?;`
const UPDATE_TEAM_IP_STATEMENT = `UPDATE teams SET container_ip = ? WHERE name = ?;`
const UPDATE_TEAM_ID_STATEMENT = `UPDATE teams SET container_id = ? WHERE name = ?;`
const UPDATE_TEAM_SCORE_STATEMENT = `UPDATE teams SET score = ? WHERE name = ?;`
const UPDATE_TEAM_UPTIME_CHECKS_STATEMENT = `UPDATE teams SET uptimeChecksTotal = ?, uptimeChecksPassed = ? WHERE name = ?;`
const UPDATE_TEAM_SERVICE_CHECKS_STATEMENT = `UPDATE teams SET serviceChecksTotal = ?, serviceChecksPassed = ? WHERE name = ?;`
const SELECT_ALL_TEAMS_STATEMENT = `SELECT name, container_ip, container_id, score, uptimeChecksTotal, uptimeChecksPassed, serviceChecksTotal, serviceChecksPassed FROM teams;`
const SELECT_ALL_TEAMS_ORDERED_STATEMENT = `SELECT name, container_ip, container_id, score, uptimeChecksTotal, uptimeChecksPassed, serviceChecksTotal, serviceChecksPassed FROM teams ORDER BY score DESC;`
const UPDATE_TEAM_STATEMENT = `UPDATE teams SET container_ip = ?, container_id = ?, score = ?, uptimeChecksTotal = ?, uptimeChecksPassed = ?, serviceChecksTotal = ?, serviceChecksPassed = ? WHERE name = ?;`

type DBTeam struct {
	Name                string `json:"name"`
	ContainerIP         string `json:"container_ip"`
	ContainerID         int    `json:"container_id"`
	Score               int    `json:"score"`
	UptimeChecksTotal   int    `json:"uptime_checks_total"`
	UptimeChecksPassed  int    `json:"uptime_checks_passed"`
	ServiceChecksTotal  int    `json:"service_checks_total"`
	ServiceChecksPassed int    `json:"service_checks_passed"`
}

func (u *DBTeam) JSON() []byte {
	json, _ := json.Marshal(u)
	return json
}

func TeamExists(name string) bool {
	rows, err := QueuedQuery(SELECT_TEAM_STATEMENT, name)

	if err != nil {
		return false
	}

	defer rows.Close()
	return rows.Next()
}

func CreateTeam(name, containerIP string, containerID, score int) (*DBTeam, error) {
	if TeamExists(name) {
		return nil, ErrTeamExists
	}

	if err := QueuedExec(INSERT_TEAM_STATEMENT, name, containerIP, containerID, score); err != nil {
		return nil, err
	}

	return GetTeam(name)
}

func GetTeam(name string) (*DBTeam, error) {
	rows, err := QueuedQuery(SELECT_TEAM_STATEMENT, name)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	if !rows.Next() {
		return nil, ErrTeamNotFound
	}

	var team DBTeam
	if err := rows.Scan(&team.Name, &team.ContainerIP, &team.ContainerID, &team.Score, &team.UptimeChecksTotal, &team.UptimeChecksPassed, &team.ServiceChecksTotal, &team.ServiceChecksPassed); err != nil {
		return nil, err
	}

	return &team, nil
}

func DeleteTeam(name string) error {
	return QueuedExec(DELETE_TEAM_STATEMENT, name)
}

func UpdateTeamIP(name, containerIP string) error {
	return QueuedExec(UPDATE_TEAM_IP_STATEMENT, containerIP, name)
}

func UpdateTeamID(name string, containerID int) error {
	return QueuedExec(UPDATE_TEAM_ID_STATEMENT, containerID, name)
}

func UpdateTeamScore(name string, score int) error {
	return QueuedExec(UPDATE_TEAM_SCORE_STATEMENT, score, name)
}

func UpdateTeamUptimeChecks(name string, total, passed int) error {
	return QueuedExec(UPDATE_TEAM_UPTIME_CHECKS_STATEMENT, total, passed, name)
}

func UpdateTeamServiceChecks(name string, total, passed int) error {
	return QueuedExec(UPDATE_TEAM_SERVICE_CHECKS_STATEMENT, total, passed, name)
}

func GetAllTeams() ([]*DBTeam, error) {
	rows, err := QueuedQuery(SELECT_ALL_TEAMS_STATEMENT)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var teams []*DBTeam
	for rows.Next() {
		var team DBTeam
		if err := rows.Scan(&team.Name, &team.ContainerIP, &team.ContainerID, &team.Score, &team.UptimeChecksTotal, &team.UptimeChecksPassed, &team.ServiceChecksTotal, &team.ServiceChecksPassed); err != nil {
			return nil, err
		}

		teams = append(teams, &team)
	}

	return teams, nil
}

func GetAllTeamsOrdered() ([]*DBTeam, error) {
	rows, err := QueuedQuery(SELECT_ALL_TEAMS_ORDERED_STATEMENT)

	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var teams []*DBTeam
	for rows.Next() {
		var team DBTeam
		if err := rows.Scan(&team.Name, &team.ContainerIP, &team.ContainerID, &team.Score, &team.UptimeChecksTotal, &team.UptimeChecksPassed, &team.ServiceChecksTotal, &team.ServiceChecksPassed); err != nil {
			return nil, err
		}

		teams = append(teams, &team)
	}

	return teams, nil
}

func UpdateTeam(team *DBTeam) error {
	return QueuedExec(UPDATE_TEAM_STATEMENT, team.ContainerIP, team.ContainerID, team.Score, team.UptimeChecksTotal, team.UptimeChecksPassed, team.ServiceChecksTotal, team.ServiceChecksPassed, team.Name)
}
