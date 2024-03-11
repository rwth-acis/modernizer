// The module 'vscode' contains the VS Code extensibility API
// Import the module and reference it with the alias vscode in your code below
import { ExtensionContext, languages, commands, Disposable, workspace} from 'vscode';
import { CodelensProvider } from './CodelensProvider';
import fetch from 'node-fetch';
import * as vscode from 'vscode';
import { DisplayVoting } from './VotingMechanism';
import { calculateURL } from './Util';

// this method is called when your extension is activated
// your extension is activated the very first time the command is executed

let disposables: Disposable[] = [];


export function activate(context: ExtensionContext) {

	const codelensProvider = new CodelensProvider();

    context.subscriptions.push(disposableGetGitURL);

	languages.registerCodeLensProvider("*", codelensProvider);

	commands.registerCommand("modernizer-vscode.enableCodeLens", () => {
		workspace.getConfiguration("modernizer-vscode").update("enableCodeLens", true, true);
	});

	commands.registerCommand("modernizer-vscode.disableCodeLens", () => {
		workspace.getConfiguration("modernizer-vscode").update("enableCodeLens", false, true);
	});

	context.subscriptions.push(disposableCustomPrompt);
	context.subscriptions.push(disposableRandomPrompt);
    context.subscriptions.push(disposableCustomPromptbyList);
}


// this method is called when your extension is deactivated
export function deactivate() {
	if (disposables) {
		disposables.forEach(item => item.dispose());
	}
	disposables = [];
}

// Helper function to get the selected function range
export function getSelectedFunctionRange(editor: vscode.TextEditor): vscode.Range | undefined {
    const selection = editor.selection;
    const document = editor.document;

    if (!selection.isEmpty) {
        // If there is a selection, use the selected range
        return new vscode.Range(selection.start, selection.end);
    } else {
        // If no selection, find the current function
        const cursorPosition = selection.start;
        let line = document.lineAt(cursorPosition);

        // Find the start of the function
        let startLine = cursorPosition.line;
        while (startLine > 0 && !line.text.trim().startsWith('func')) {
            startLine--;
            line = document.lineAt(startLine);
        }

        // Find the end of the function
        let endLine = cursorPosition.line;
        while (endLine < document.lineCount - 1 && !line.text.trim().endsWith('}')) {
            endLine++;
            line = document.lineAt(endLine);
        }

        // Return the range of the found function
        return new vscode.Range(startLine, 0, endLine, line.text.length);
    }
}

let disposableRandomPrompt = vscode.commands.registerCommand('modernizer-vscode.randomPrompt', async () => {
	try {
		await generateRandomPrompt();
	} catch (error: any) {
		vscode.window.showErrorMessage(`Error: ${error.message}`);
	}
});

let disposableCustomPrompt = vscode.commands.registerCommand('modernizer-vscode.customPrompt', async () => {
	try {
		await generateCustomPrompt();
	} catch (error: any) {
		vscode.window.showErrorMessage(`Error: ${error.message}`);
	}
});

let disposableGetGitURL = vscode.commands.registerCommand('modernizer-vscode.gitURL', async () => {
	try {
		let URL = await calculateURL();
        vscode.window.showInformationMessage(`Notification: ${URL}`);
	} catch (error: any) {
		vscode.window.showErrorMessage(`Error: ${error.message}`);
	}
});

let disposableCustomPromptbyList = vscode.commands.registerCommand('modernizer-vscode.PromptByList', async () => {
	try {
		await generateCustomPromptbyList();
	} catch (error: any) {
		vscode.window.showErrorMessage(`Error: ${error.message}`);
	}
});

async function generateRandomPrompt() {
    const activeEditor = vscode.window.activeTextEditor;
    if (!activeEditor) {
        vscode.window.showWarningMessage('No active text editor found.');
        return;
    }

    const gitURL = await calculateURL();

    // Get the selected function range
    const selectedFunctionRange = getSelectedFunctionRange(activeEditor);
    if (!selectedFunctionRange) {
        vscode.window.showWarningMessage('No function selected. Please select a function to generate a prompt.');
        return;
    }

    // Extract the function code from the range
    const functionCode = activeEditor.document.getText(selectedFunctionRange);

    const baseUrl: string = vscode.workspace.getConfiguration("modernizer-vscode").get("baseURL", "https://modernizer.milki-psy.dbis.rwth-aachen.de");
    const generateRoute: string = '/generate';
    const url: string = `${baseUrl}${generateRoute}`;

    const promptData = {
        model: 'codellama:34b-instruct',
        prompt: `${functionCode}`,
        gitURL: gitURL,
    };

    await sendPromptToAPI(url, promptData);
}

async function generateCustomPrompt() {
    const activeEditor = vscode.window.activeTextEditor;
    if (!activeEditor) {
        vscode.window.showErrorMessage("No text selected.");
        return;
    }

    const gitURL = await calculateURL();

    const selectedFunctionRange = getSelectedFunctionRange(activeEditor);
    if (!selectedFunctionRange) {
        vscode.window.showWarningMessage('No function selected. Please select a function to generate a prompt.');
        return;
    }

    const functionCode = activeEditor.document.getText(selectedFunctionRange);

    const userInput = await vscode.window.showInputBox({
        prompt: "Enter instruct to use in prompt"
    });

    if (!userInput) return; // User canceled input

    const baseUrl: string = vscode.workspace.getConfiguration("modernizer-vscode").get("baseURL", "https://modernizer.milki-psy.dbis.rwth-aachen.de");
    const generateRoute: string = "/generate";
    const url: string = `${baseUrl}${generateRoute}`;

    const promptData = {
        model: "codellama:34b-instruct",
        prompt: `${functionCode}`,
        instruct: userInput,
        gitURL: gitURL,
    };

    await sendPromptToAPI(url, promptData);
}

async function generateCustomPromptbyList() {
    const activeEditor = vscode.window.activeTextEditor;
    if (!activeEditor) {
        vscode.window.showErrorMessage("No text selected.");
        return;
    }

    const gitURL = await calculateURL();

    const selectedFunctionRange = getSelectedFunctionRange(activeEditor);
    if (!selectedFunctionRange) {
        vscode.window.showWarningMessage('No function selected. Please select a function to generate a prompt.');
        return;
    }

    const functionCode = activeEditor.document.getText(selectedFunctionRange);


    const selectedInstruct = await showInstructTemplates();

    if (!selectedInstruct) return; // User canceled input

    const baseUrl: string = vscode.workspace.getConfiguration("modernizer-vscode").get("baseURL", "https://modernizer.milki-psy.dbis.rwth-aachen.de");
    const generateRoute: string = "/generate";
    const url: string = `${baseUrl}${generateRoute}`;

    const promptData = {
        model: "codellama:34b-instruct",
        prompt: `${functionCode}`,
        instruct: selectedInstruct,
        gitURL: gitURL,

    };

    await sendPromptToAPI(url, promptData);
}

async function sendPromptToAPI(url: string, promptData: any) {
    return vscode.window.withProgress({
        location: vscode.ProgressLocation.Notification,
        title: 'Sending prompt to API...',
        cancellable: false
    }, async (progress, token) => {
        try {
            const response = await fetch(url, {
                method: "POST",
                headers: {
                    "Content-Type": "application/json"
                },
                body: JSON.stringify(promptData)
            });

            if (!response.ok) {
                throw new Error(`Failed to make request: ${response.statusText}`);
            }

            const contentType = response.headers.get("content-type");
            if (!contentType || !contentType.includes("application/json")) {
                throw new Error("Received non-JSON response. Check the API for possible errors.");
            }

            const responseBody = await response.json();
            const responseText = responseBody.response || "No response field found";

            const outputWindow = vscode.window.createOutputChannel("Ollama Response");
            outputWindow.show(true);

            outputWindow.append(`Generated new response with the instruct: ${responseBody.instruct}\n\n`);
            outputWindow.append(responseText + "\n");

            if (responseBody.promptID) {
                DisplayVoting(responseBody.promptID);
            } else {
                vscode.window.showWarningMessage("No promptId field found");
            }
        } catch (error: any) {
            vscode.window.showErrorMessage(`Error: ${error.message}`);
        }
    });
}



async function showInstructTemplates(): Promise<string | undefined> {
    let baseUrl: string = vscode.workspace.getConfiguration("modernizer-vscode").get("baseURL", "https://modernizer.milki-psy.dbis.rwth-aachen.de");
    let responseListPath: string = '/get-all-sets';
    let url: string = `${baseUrl}${responseListPath}`;
    let result: string | undefined;

    try {
        const response = await fetch(url);
        const sets = await response.json();

        result = await vscode.window.showQuickPick(sets, {
            placeHolder: 'Select a set',
        });
    } catch (error: any) {
        vscode.window.showErrorMessage(`Error: ${error.message}`);
        return ''; // Return an empty string in case of an error
    }

    baseUrl = vscode.workspace.getConfiguration("modernizer-vscode").get("baseURL", "https://modernizer.milki-psy.dbis.rwth-aachen.de");
    responseListPath = '/get-instruct';
    url = `${baseUrl}${responseListPath}`;

    let queryParams = new URLSearchParams(result ? { set: result } : {});
    let urlQuery = `${url}?${queryParams.toString()}&all=true`;

    const response = await fetch(urlQuery);
    if (!response.ok) {
        vscode.window.showErrorMessage(`Error: ${response.statusText}`);
        return '';
    }

    try {
        const instructs = await response.json();
        result = await vscode.window.showQuickPick(instructs.result, {
            placeHolder: 'Select an instruct',
        });

        return result;
    } catch (error) {
        console.error("Error parsing response data:", error);
        return "";
    }
}
