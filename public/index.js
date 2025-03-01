import * as api from "./lib/api.js";

if (window.matchMedia("(prefers-color-scheme: dark)").matches) {
    document.documentElement.classList.add("dark");
}

const accessButton = document.querySelector("button#accessButton");
const loginButton = document.querySelector("button#loginButton");
const loginModal = document.querySelector("div#login");
const mainContent = document.querySelector("div#main");
const topnav = document.querySelector("div#topnav");
const containerTemplate = document.querySelector("template#containerTemplate");
const createContainerDropdown = document.querySelector("div#createContainerDropdown");

let isAuthenticated = await api.pollAccess();

function updateDisplaysBasedOnAccess() {
    if (isAuthenticated) {
        // Login Button -> Logout Button
        accessButton.classList.add("logout");
        accessButton.textContent = "Logout";

        // Show Create Container Button
        createContainerDropdown.classList.remove("hidden");

        // Show dropdowns on containers
        document.querySelectorAll("div.container").forEach(container => container.querySelector("div.containerDropdown").classList.remove("hidden"));
    } else {
        // Logout Button -> Login Button
        accessButton.classList.remove("logout");
        accessButton.textContent = "Login";

        // Hide Create Container Button
        createContainerDropdown.classList.add("hidden");

        // Hide dropdowns on containers
        document.querySelectorAll("div.container").forEach(container => container.querySelector("div.containerDropdown").classList.add("hidden"));
    }
}

accessButton.addEventListener("click", function () {
    if (isAuthenticated) {
        isAuthenticated = false;
        api.logout();
        updateDisplaysBasedOnAccess();
        loginModal.querySelector("#username").value = "";
        loginModal.querySelector("#password").value = "";
        return;
    }

    loginModal.classList.remove("hidden");
    topnav.classList.add("hidden");
    mainContent.classList.add("hidden");
});

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
        loginModal.classList.add("hidden");
        topnav.classList.remove("hidden");
        mainContent.classList.remove("hidden");

        updateDisplaysBasedOnAccess();
    } else {
        alert("Login failed: " + response.statusText);
    }
}

loginButton.addEventListener("click", function () {
    const username = document.getElementById("username").value;
    const password = document.getElementById("password").value;

    if (username && password) {
        tryLogin(username, password);
    } else {
        alert("Please enter both username and password.");
    }
});

/** @param {api.APIContainer} apiContainer */
function createNewContainerElement(apiContainer) {
    /** @type {HTMLDivElement} */
    const container = containerTemplate.content.cloneNode(true).querySelector("div.container");

    container.dataset.id = apiContainer.team.name;

    container.querySelector(".containerName").textContent = apiContainer.team.name;
    container.querySelector(".containerStatus").classList.add("status-" + (apiContainer.team.checks.failed > 0 ? "services-down" : apiContainer.container.status));

    container.querySelector("span.containerPVE").textContent = "CT-" + apiContainer.container.pve_id;
    container.querySelector("span.containerIPv4").textContent = apiContainer.container.ipv4;
    container.querySelector("span.containerIPv4").onclick = () => window.open("http://" + apiContainer.container.ipv4, "_blank");
    container.querySelector("span.containerScore").textContent = apiContainer.team.score;
    container.querySelector("span.containerUptime").textContent = (apiContainer.team.uptime * 100).toFixed(2) + "%";
    container.querySelector("span.containerServiceChecks").textContent = apiContainer.team.checks.passed + "/" + apiContainer.team.checks.total;
    container.querySelector("div.containerDropdown").classList[isAuthenticated ? "remove" : "add"]("hidden");

    mainContent.appendChild(container);
}

/** @param {api.APIContainer} apiContainer */
function updateContainerElement(apiContainer) {
    const existingContainer = mainContent.querySelector(`div.container[data-id="${apiContainer.team.name}"]`);

    if (!existingContainer) {
        createNewContainerElement(apiContainer);
    } else {
        existingContainer.querySelector(".containerStatus").classList.add("status-" + (apiContainer.team.checks.failed > 0 ? "services-down" : apiContainer.container.status));
        existingContainer.querySelector("span.containerScore").textContent = apiContainer.team.score;
        existingContainer.querySelector("span.containerUptime").textContent = (apiContainer.team.uptime * 100).toFixed(2) + "%";
        existingContainer.querySelector("span.containerServiceChecks").textContent = apiContainer.team.checks.passed + "/" + apiContainer.team.checks.total;
    }
}

async function updateInterval() {
    const response = await api.getContainers();

    if (!response.ok) {
        alert("Failed to fetch containers: " + response.statusText);
        return;
    }

    /** @type {api.APIContainer[]} */
    const containers = response.data;
    containers.forEach(updateContainerElement);

    const topData = document.querySelector("span#topStatus");
    const bestContainer = containers.reduce((best, current) => {
        if (!best || current.team.score > best.team.score) {
            return current;
        }
        return best;
    }, null);

    topData.textContent = `${containers.filter(c => c.container.status === "running").length}/${containers.length} containers running, ${bestContainer.team.name} is in the lead with ${bestContainer.team.score} points`;

    setTimeout(updateInterval, 5000);
}

updateInterval();
updateDisplaysBasedOnAccess();