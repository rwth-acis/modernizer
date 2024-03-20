// The module 'vscode' contains the VS Code extensibility API
// Import the module and reference it with the alias vscode in your code below
import { ExtensionContext, languages, commands, Disposable, workspace} from 'vscode';
import { CodelensProvider } from './CodelensProvider';
import fetch from 'node-fetch';
import * as vscode from 'vscode';
import { DisplayVoting } from './VotingMechanism';
import { calculateURL } from './Util';
import { GetResponseListType } from './CodelensProvider';

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
    context.subscriptions.push(disposableGetSimilarCode);
    context.subscriptions.push(disposableSavePrompt);
    context.subscriptions.push(disposableGetResponseByType);
    context.subscriptions.push(disposableGetSimilarMeaning);
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

let disposableRandomPrompt = vscode.commands.registerCommand('modernizer-vscode.randomExplanationPrompt', async () => {
	try {
		await randomExplanationPrompt();
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
        vscode.window.showInformationMessage(`No similar code found.`);
	}
});

let disposableGetSimilarCode = vscode.commands.registerCommand('modernizer-vscode.getSimilarCode', async () => {
    try {
        const gitURLs: string[] = await GetSimilarCode();
        displayGitURLs(gitURLs);
    } catch (error: any) {
        vscode.window.showInformationMessage(`No similar code found.`);
    }
});

let disposableGetSimilarMeaning = vscode.commands.registerCommand('modernizer-vscode.getSimilarMeaning', async () => {
    try {
        const gitURLs: string[] = await GetSimilarMeaning();
        displayGitURLs(gitURLs);
    } catch (error: any) {
        vscode.window.showInformationMessage(`No similar code found.`);
    }
});

let disposableGetResponseByType = vscode.commands.registerCommand('modernizer-vscode.getResponseType', async () => {
	try {
		await showResponseByType();
	} catch (error: any) {
		vscode.window.showErrorMessage(`Error: ${error.message}`);
	}
});

let disposableSavePrompt = vscode.commands.registerCommand('modernizer-vscode.savePrompt', async () => {
        // Get the active text editor
        const editor = vscode.window.activeTextEditor;
        if (!editor) {
            vscode.window.showErrorMessage('No active text editor found.');
            return;
        }
    
        const outputWindowContent = editor.document.getText();
    
        const instructRegex = /Generated new response with the custom instruct: (.*)/g;
        const match = instructRegex.exec(outputWindowContent);
        if (!match || !match[1]) {
            vscode.window.showErrorMessage('No instruct found in the output window.');
            return;
        }
        const instruct = match[1];
    
        const customSet: string[] = [instruct];
        await setCustomSet(customSet);
    
        vscode.window.showInformationMessage('Custom prompt saved successfully!');
});

async function randomExplanationPrompt() {
    const activeEditor = vscode.window.activeTextEditor;
    if (!activeEditor) {
        vscode.window.showWarningMessage('No active text editor found.');
        return;
    }

    const gitURL = await calculateURL();

    const selectedFunctionRange = getSelectedFunctionRange(activeEditor);
    if (!selectedFunctionRange) {
        vscode.window.showWarningMessage('No function selected. Please select a function to generate a prompt.');
        return;
    }

    const functionCode = activeEditor.document.getText(selectedFunctionRange);

    const promptData = {
        model: 'codellama:34b-instruct',
        prompt: `${functionCode}`,
        gitURL: gitURL,
        instructType: 'explanation'
    };

    await sendPromptToAPI(promptData, false);
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

    const promptData = {
        model: "codellama:34b-instruct",
        prompt: `${functionCode}`,
        instruct: userInput,
        gitURL: gitURL,
        instructType: 'custom'
    };

    await sendPromptToAPI(promptData, true);
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


    const promptData = {
        model: "codellama:34b-instruct",
        prompt: `${functionCode}`,
        instructType: selectedInstruct[0],
        instruct: selectedInstruct[1],
        gitURL: gitURL,
    };

    await sendPromptToAPI(promptData, false);
}

async function sendPromptToAPI(promptData: any, custom: boolean){

    const baseUrl: string = vscode.workspace.getConfiguration("modernizer-vscode").get("baseURL", "https://modernizer.milki-psy.dbis.rwth-aachen.de");
    const generateRoute: string = "/generate";
    const url: string = `${baseUrl}${generateRoute}`;

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

            OutputWindowGeneratedResponse(responseBody.instruct, responseText, responseBody.promptID, custom);

        } catch (error: any) {
            vscode.window.showErrorMessage(`Error: ${error.message}`);
        }
    });
}

async function OutputWindowGeneratedResponse(instruct: string, response: string, ID: string, custom: boolean) {

    const outputWindow = vscode.window.createOutputChannel("Ollama Response", "plaintext");
    outputWindow.show(true);

    if (custom) {
        outputWindow.append(`Generated new response with the custom instruct: ${instruct}\n\n`);
    } else {
        outputWindow.append(`Generated new response with the instruct: ${instruct}\n\n`);
    }
    outputWindow.append(response + "\n");


    DisplayVoting(ID);
}



async function showInstructTemplates(): Promise<string[] | undefined> {
    let baseUrl: string = vscode.workspace.getConfiguration("modernizer-vscode").get("baseURL", "https://modernizer.milki-psy.dbis.rwth-aachen.de");
    let responseListPath: string = '/get-all-sets';
    let selectedSet: string[] | undefined;

    try {
        const response = await fetch(`${baseUrl}${responseListPath}`);
        const sets = await response.json();

        sets.sort();
        sets.unshift("Custom");

        const set = await vscode.window.showQuickPick(sets, {
            placeHolder: 'Select a set',
            canPickMany: false
        });

        selectedSet = set ? [set] : undefined;
    } catch (error: any) {
        vscode.window.showErrorMessage(`Error: ${error.message}`);
        return undefined; // Return undefined in case of an error
    }

    if (!selectedSet || selectedSet.length === 0) {
        return undefined;
    }

    if (selectedSet[0] === "Custom") {
        let instructs =  getCustomSet();
        instructs.sort();
        const selectedInstruct = await vscode.window.showQuickPick(instructs, {
            placeHolder: 'Select an instruct',
            canPickMany: false
        });

        return selectedInstruct ? [selectedSet[0], selectedInstruct] : undefined;
    }

    baseUrl = vscode.workspace.getConfiguration("modernizer-vscode").get("baseURL", "https://modernizer.milki-psy.dbis.rwth-aachen.de");
    responseListPath = '/get-instruct';
    const selectedSetName = selectedSet[0];
    const queryParams = new URLSearchParams(selectedSetName ? { set: selectedSetName } : {});
    const urlQuery = `${baseUrl}${responseListPath}?${queryParams.toString()}&all=true`;

    const response = await fetch(urlQuery);
    if (!response.ok) {
        vscode.window.showErrorMessage(`Error: ${response.statusText}`);
        return undefined;
    }

    try {
        const instructs = await response.json();
        instructs.result.sort();
        const selectedInstruct = await vscode.window.showQuickPick(instructs.result, {
            placeHolder: 'Select an instruct',
            canPickMany: false // Ensuring only one selection at a time
        });

        return selectedInstruct ? [selectedSet[0], selectedInstruct] : undefined;
    } catch (error) {
        console.error("Error parsing response data:", error);
        return undefined;
    }
}

function getCustomSet(): string[] {
    const customSet: string[] = vscode.workspace.getConfiguration("modernizer-vscode").get("customSet", []);
    return customSet;
}
async function setCustomSet(customSet: string[]): Promise<void> {
    const configuration = vscode.workspace.getConfiguration("modernizer-vscode");
    let currentCustomSet = configuration.get<string[]>("customSet", []);
    const updatedCustomSet = Array.from(new Set([...currentCustomSet, ...customSet]));

    await configuration.update("customSet", updatedCustomSet, vscode.ConfigurationTarget.Global);
}

async function GetSimilarCode(): Promise<string[]> {
    try {
        const activeEditor = vscode.window.activeTextEditor;
        if (!activeEditor) {
            return [];
        }

        const selectedFunctionRange = getSelectedFunctionRange(activeEditor);
        const functionCode = activeEditor.document.getText(selectedFunctionRange);

        
        const baseUrl: string = vscode.workspace.getConfiguration("modernizer-vscode").get("baseURL", "https://modernizer.milki-psy.dbis.rwth-aachen.de");
        const path: string = '/get-similar-code';
        const url: string = `${baseUrl}${path}`;

        const queryParams = new URLSearchParams({ code: functionCode });
        const urlQuery = `${url}?${queryParams.toString()}`;

        const response = await fetch(urlQuery);
        if (!response.ok) {
            throw new Error("Failed to fetch data");
        }

        const data = await response.json();
        return data as string[];
    } catch (error: any) {
        throw new Error("Failed to retrieve data: " + error.message);
    }
}

async function GetSimilarMeaning(): Promise<string[]> {

    const userInput = await vscode.window.showInputBox({
        prompt: "Enter the semantic meaning you are looking for"
    });

    if (!userInput) {
        return [];
    }

    const baseUrl: string = vscode.workspace.getConfiguration("modernizer-vscode").get("baseURL", "https://modernizer.milki-psy.dbis.rwth-aachen.de");
    const path: string = '/get-similar-meaning';
    const url: string = `${baseUrl}${path}`;
    const queryParams = new URLSearchParams({ meaning: userInput });
    const urlQuery = `${url}?${queryParams.toString()}`;

    try {
        const response = await fetch(urlQuery);
        if (!response.ok) {
            throw new Error("Failed to fetch data");
        }

        const data = await response.json();
        return data as string[];
    } catch (error: any) {
        throw new Error("Failed to retrieve data: " + error.message);
    }
}


async function displayGitURLs(gitURLs: string[]): Promise<void> {
    const outputChannel = vscode.window.createOutputChannel('Similar Git URLs');
    outputChannel.show();
    outputChannel.appendLine('Similar Code can be found in the following Git URLs:\n\n');

    let URL = await calculateURL();

    gitURLs = gitURLs.filter(gitURL => gitURL !== URL);
    const uniqueURLs = [...new Set(gitURLs)];
    
    uniqueURLs.forEach((gitURL, index) => {
        outputChannel.appendLine(`URL ${index + 1}: ${gitURL}`);
    });
}

async function showResponseByType() {
    try {
        const activeEditor = vscode.window.activeTextEditor;
        if (!activeEditor) {
            return;
        }

        const selectedFunctionRange = getSelectedFunctionRange(activeEditor);
        const functionCode = activeEditor.document.getText(selectedFunctionRange);

        const types = await getTypes(functionCode);
        if (types.length === 0) {
            vscode.window.showErrorMessage("No response found.");
            return;
        }

            const set = await vscode.window.showQuickPick(types, {
                placeHolder: 'Select a set',
                canPickMany: false
            });

            await GetResponseListType(functionCode, set ?? ""); 

     } catch (error: any) {
            vscode.window.showErrorMessage("Error: " + error.message);
     }

}



async function getTypes(code: string): Promise<string[]> {
    const baseUrl: string = vscode.workspace.getConfiguration("modernizer-vscode").get("baseURL", "https://modernizer.milki-psy.dbis.rwth-aachen.de");
    const path: string = '/get-instructtype';
    const url: string = `${baseUrl}${path}`;

    const queryParams = new URLSearchParams({ code: code });
    const urlQuery = `${url}?${queryParams.toString()}`;

    const response = await fetch(urlQuery);
    if (!response.ok) {
        return [];
    }

    try {
        const responseData = await response.json();
        return responseData;
    } catch (error) {
        console.error("Error parsing response data:", error);
        return [];
    }
}