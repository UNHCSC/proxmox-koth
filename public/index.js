import * as api from "./lib/api.js";

const loginButton = document.querySelector("button#loginButton");
const loginModal = document.querySelector("div#login");
const mainContent = document.querySelector("div#main");
const topnav = document.querySelector("div#topnav");
const containerTemplate = document.querySelector("template#containerTemplate");

let isAuthenticated = await api.pollAccess();

async function tryLogin(username, password) {
    const response = await fetch("/api/login", {
        method: "POST",
        headers: {
            "Content-Type": "text/plain",
        },
        body: JSON.stringify({ username, password }),
    });

    if (response.ok) {
        isAuthenticated = true;
        switchToMain();
    } else {
        alert("Login failed: " + response.statusText);
    }
}

if (isAuthenticated) {
    switchToMain();
} else {
    loginButton.addEventListener("click", function () {
        const username = document.getElementById("username").value;
        const password = document.getElementById("password").value;
    
        if (username && password) {
            tryLogin(username, password);
        } else {
            alert("Please enter both username and password.");
        }
    });
}

/** @param {api.APIContainer} apiContainer */
function createNewContainerElement(apiContainer) {
    /** @type {HTMLDivElement} */
    const container = containerTemplate.content.cloneNode(true).querySelector("div.container");

    container.querySelector(".containerName").textContent = apiContainer.team.name;
    container.querySelector(".containerStatus").classList.add("status-" + apiContainer.container.status);

    container.querySelector("span.containerPVE").textContent = "CT-" + apiContainer.container.pve_id;
    container.querySelector("span.containerIPv4").textContent = apiContainer.container.ipv4;
    container.querySelector("span.containerIPv4").onclick = function () {
        // navigator.clipboard.writeText(apiContainer.container.ipv4).then(() => {
        //     alert("IPv4 address copied to clipboard: " + apiContainer.container.ipv4);
        // }).catch(err => {
        //     console.error("Failed to copy IPv4 address: ", err);
        // });

        window.open("http://" + apiContainer.container.ipv4, "_blank");
    }
    container.querySelector("span.containerScore").textContent = apiContainer.team.score;
    container.querySelector("span.containerUptime").textContent = (apiContainer.team.uptime * 100).toFixed(2) + "%";
    container.querySelector("span.containerServiceChecks").textContent = apiContainer.team.checks.passed + "/" + apiContainer.team.checks.total;

    mainContent.appendChild(container);
}

async function switchToMain() {
    loginModal.classList.add("hidden");
    topnav.classList.remove("hidden");
    mainContent.classList.remove("hidden");

    const response = await api.getContainers();

    if (!response.ok) {
        alert("Failed to fetch containers: " + response.statusText);
        return;
    }

    /** @type {api.APIContainer[]} */
    const containers = response.data;
    containers.forEach(createNewContainerElement);

    const topData = document.querySelector("span#topStatus");
    const bestContainer = containers.reduce((best, current) => {
        if (!best || current.team.score > best.team.score) {
            return current;
        }
        return best;
    }, null);

    topData.textContent = `${containers.filter(c => c.container.status === "running").length}/${containers.length} containers running, ${bestContainer.team.name} is in the lead with ${bestContainer.team.score} points`;
}