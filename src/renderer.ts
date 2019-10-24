import "semantic-ui-css/semantic.min.css"; // I don't even know... without it the LOCAL FILES would not properly get loaded
const { ipcRenderer } = nodeRequire('electron')
const base64url = nodeRequire('base64url');

// import "semantic-ui";
// const $: JQueryStatic = (window as any)["jQuery"];

const localhost: string = document.location.host || "localhost";
const fetchMsg = JSON.stringify({
	fetch: {},
});

const getRecipientsMsg = JSON.stringify({
	clients: {},
});

const getDetailsMsg = JSON.stringify({
	details: {},
});



// doesn't have any fancy signatures, etc because I WON'T do elliptic crypto in js unless
// absolutely neccessary
interface ElectronChatMessage {
	content: string;
	senderPublicKey: string;
	senderProviderPublicKey: string;
}

interface ClientData {
	id: string;
	pubKey: string;
	provider: {
		id: string;
		host: string;
		port: string;
		pubKey: string;
	};
}

class SocketConnection {
	private conn: WebSocket;
	private ticker: number;
	private clients: ClientData[];
	private ownDetails: ClientData;
	constructor(port: string) {
		const conn = new WebSocket(`ws://${localhost}:${port}/mix`);
		conn.onclose = this.onSocketClose;
		conn.onmessage = this.onSocketMessage;
		conn.onerror = (ev: Event) => console.log("Socket error: ", ev);
		conn.onopen = this.onSocketOpen.bind(this);

		this.clients = [];
		this.conn = conn;
		this.ticker = window.setInterval(this.fetchMessages.bind(this), 1000);
	}

	public closeConnection() {
		this.conn.close();
		window.clearInterval(this.ticker);
	}

	public getClients() {
		console.log("getting list of available clients...");
		this.conn.send(getRecipientsMsg);
	}

	public sendMessage(message: string) {
		const selectedRecipientIdx = $("#recipientSelector").dropdown("get value");
		if (selectedRecipientIdx.length === 0) {
			return;
		}

		const selectedRecipient = this.clients[selectedRecipientIdx];

		// once recipient is selected, don't allow changing it
		if (!$("#recipientSelector").hasClass("disabled")) {
			$("#recipientSelector").addClass("disabled");
			// also update the sender divider here
			updateSenderDivider(formatDisplayedClient(selectedRecipient));
		}

		console.log(selectedRecipient);

		const fullMessage: ElectronChatMessage = {
			content: message,
			senderProviderPublicKey: this.ownDetails.provider.pubKey,
			senderPublicKey: this.ownDetails.pubKey,
		};

		const sendMsg = JSON.stringify({
				send: {
					message: btoa(JSON.stringify(fullMessage)),
					recipient: selectedRecipient,
				},
		});

		this.conn.send(sendMsg);
		createChatMessage("you", message, true);
	}

	private onSocketOpen() {
		this.getClients();
		this.conn.send(getDetailsMsg);
	}

	private onSocketClose(ev: CloseEvent) {
		console.log("The websocket was closed", ev);

		const innerHeader = $("<div>", {
			class: "sub header",
			id: "wsclosedsub",
		}).text(`(code: ${ev.code}) - ${ev.reason || "no reason provided"}`);

		const contentDiv = $("<div>", {
			class: "content",
			id: "wsclosedcontent",
		}).text("The websocket was closed.").append(innerHeader);

		const closedIcon = $("<i>", {
			class: "close icon",
			id: "wsclosedicon",
		});

		const closedHeader = $("<h2>", {
			class: "ui icon header",
			id: "wsclosedheader",
		}).append(closedIcon, contentDiv);

		$("#noticeDiv").append(closedHeader);
	}

	private handleFetchResponse(fetchData: any) {
		const messages = fetchData.fetch.messages;

		for (const rawMsg of messages) {
			// TODO: FIXME: for some reason at some point there's an invalid character attached here with ascii code 1...
			let b64Decoded: string = base64url.decode(rawMsg);
			if (b64Decoded.charCodeAt(0) < 32) {
				b64Decoded = b64Decoded.substring(1);
			}
			const msg = JSON.parse(b64Decoded) as ElectronChatMessage;

			createChatMessage(
				`??? - ${msg.senderPublicKey.substring(0, 8)}... (Provider: ${msg.senderProviderPublicKey.substring(0, 8)}...)`,
				msg.content,
			);
		}
	}

	private handleClientsResponse(clientsData: any) {
		if ($("#recipientSelector").hasClass("disabled")) {
			$("#recipientSelector").removeClass("disabled");
		}

		const availableClients = clientsData.clients.clients as ClientData[];
		// update our current list
		this.clients = availableClients;

		const valuesArray = availableClients.map((client, idx) => {
			return {name: formatDisplayedClient(client), value: idx};
		});

		$("#recipientSelector").dropdown({
			placeholder: "Choose recipient",
			fullTextSearch: true,
			values: valuesArray, // don't mind the errors, it's just typescript not really liking jquery
		});
	}

	private displayOwnDetails(ownDetails: ClientData) {
		let detailsString = "Your public key is: " + ownDetails.pubKey;
		detailsString += "\n Your provider's public key is: " + ownDetails.provider.pubKey;
		createChatMessage("SYSTEM INFO", detailsString, true);
	}
	// had to define it as an arrow function otherwise I couldn't call this.handle...
	private onSocketMessage = (ev: MessageEvent): void => {

		// we can either receive list of clients or actual message
		const receivedData = JSON.parse(ev.data);

		if (receivedData.hasOwnProperty("fetch")) {
			return this.handleFetchResponse(receivedData);
		} else if (receivedData.hasOwnProperty("clients")) {
			return this.handleClientsResponse(receivedData);
		} else if (receivedData.hasOwnProperty("details")) {
			this.ownDetails = receivedData.details.details;
			// fix up encoding
			this.ownDetails.id = base64url.fromBase64(this.ownDetails.id);
			this.ownDetails.pubKey = base64url.fromBase64(this.ownDetails.pubKey);
			this.ownDetails.provider.id = base64url.fromBase64(this.ownDetails.provider.id);
			this.ownDetails.provider.pubKey = base64url.fromBase64(this.ownDetails.provider.pubKey);

			return this.displayOwnDetails(this.ownDetails);
		} else if (receivedData.hasOwnProperty("send")) {
			console.log("received send confirmation");
		}

		console.log("Received unknown response!");
		console.log(receivedData);
	}

	private fetchMessages() {
		console.log("checking for new messages...");
		this.conn.send(fetchMsg);
	}
}

function updateSenderDivider(displayedName: string) {
	$("#senderDivider").html("Sending to " + displayedName);
}

function formatDisplayedClient(client: ClientData): string {
	return "??? - " + client.id.substring(0, 8) + "...";
}

function createChatMessage(senderID: string, content: string, isReply: boolean = false) {
	const textDiv = $("<div>", {
		class: "text",
	}).text(content);

	// TODO: we really should get it from message itself rather than use local time, but oh well
	const dateDiv = $("<div>", {
		class: "date",
	}).text(new Date().toLocaleTimeString());

	const metadataDiv = $("<div>", {
		class: "metadata",
	}).append(dateDiv);

	const authorAnchor = $("<a>", {
		class: "author",
	}).text(senderID);

	const contentDiv = $("<div>", {
		class: "content",
	}).append(authorAnchor, metadataDiv, textDiv);

	const avatarAnchor = $("<a>", {
		class: "avatar",
	}).html('<img src="assets/avatar.png">');

	const commentDiv = $("<div>", {
		class: "comment",
	}).append(avatarAnchor, contentDiv);

	let chatMessageDiv: JQuery<HTMLElement>;
	if (isReply) {
		chatMessageDiv = $("<div>", {
			class: "chatMessage reply",
		});
	} else {
		chatMessageDiv = $("<div>", {
			class: "chatMessage incoming",
		});
	}
	chatMessageDiv.append(commentDiv);

	$("#messagesList").append(chatMessageDiv);
}

function handleSendAction(conn: SocketConnection) {
	const inputElement = $("#msgInput");
	const messageInput = inputElement.val() as string;
	if (messageInput.length === 0) {
		return; // don't do anything if there's nothing to send
	}

	conn.sendMessage(messageInput);

	// finally clear input box
	inputElement.val("");
}

function main() {
	const port: string = ipcRenderer.sendSync("port");

	let conn = new SocketConnection(port);
	$("#closeWS").click(() => {
		conn.closeConnection();
	});
	$("#remakeWS").click(() => {
		conn = new SocketConnection(port);
		$("#noticeDiv").html("");
	});
	$("#getClients").click(() => {
		conn.getClients();
	});
	$("#sendBtn").click(() => {
		handleSendAction(conn);
	});
	$("#msgInput").on("keydown", (ev: JQuery.KeyDownEvent) => {
		if (ev.keyCode === 13) {
			handleSendAction(conn);
		}
	});
}

$(document).ready(() => main());
