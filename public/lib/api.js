export const POST_FIELDS = {
    method: "POST",
    headers: {
        "Content-Type": "text/plain"
    },
    credentials: "include"
};

export class APIResponse {
    /**
     * A cache of HTTPCat images
     * @type {Map<number, HTMLImageElement>}
     */
    static catCache = new Map();

    /**
     * An API response object
     * @param {number} status The HTTP status code
     * @param {any} data The response data
     */
    constructor(status, data) {
        this.status = status;
        this.data = data;
    }

    /**
     * Get the status name of the response
     * @returns {string} The status name
     */
    getStatusName() {
        return ({
            200: "OK",
            201: "Created",
            226: "IM Used",
            400: "Bad Request",
            401: "Unauthorized",
            403: "Forbidden",
            404: "Not Found",
            405: "Method Not Allowed",
            406: "Not Acceptable",
            415: "Unsupported Media Type",
            500: "Internal Server Error",
            501: "Not Implemented",
            502: "Bad Gateway",
            503: "Service Unavailable",
        }[this.status] || "Unknown Status");
    }

    /**
     * Get the HTTPCat image for the response
     * @returns {Promise<HTMLImageElement>}
     */
    async getHTTPCat() {
        if (APIResponse.catCache.has(this.status)) {
            return APIResponse.catCache.get(this.status);
        }

        const image = new Image();

        await new Promise((resolve, reject) => {
            image.onload = resolve;
            image.onerror = reject;
            image.src =
                this.getStatusName() === "Unknown Status" ?
                    "https://http.cat/418.jpg" :
                    `https://http.cat/${this.status}.jpg`;
        });

        APIResponse.catCache.set(this.status, image);
        return image;
    }

    /**
     * @type {boolean}
     */
    get ok() {
        return this.status === 200;
    }
}

export class APIContainer {
    static statuses = ["running", "stopped", "unknown"];

    container = {
        ipv4: "0.0.0.0",
        pve_id: 0,
        status: "unknown"
    };

    team = {
        name: "unknown",
        score: 0,
        uptime: 0,
        checks: {
            failed: 0,
            passed: 0,
            total: 0
        }
    };

    lastUpdate = new Date().getTime();
}

export async function getContainers() {
    const response = await fetch("/api/public/summary.json");

    return new APIResponse(response.status, response.status === 200 ? await response.json() : null);
}

export async function pollAccess() {
    return (await fetch("/api/checkLogin", {
        credentials: "include"
    })).status === 200;
}

export async function logout() {
    const response = await fetch("/api/logout", {
        method: "DELETE",
        credentials: "include"
    });

    return new APIResponse(response.status, null);
}

export async function createContainer(teamName, ipAddress) {
    const response = await fetch("/api/create", {
        method: "POST",
        credentials: "include",
        headers: {
            "Content-Type": "text/plain"
        },
        body: JSON.stringify({
            name: teamName,
            ip: ipAddress
        })
    });

    return new APIResponse(response.status, response.status === 200 ? null : await response.text());
}
