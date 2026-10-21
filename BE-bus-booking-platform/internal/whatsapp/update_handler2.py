import re

def find_matching_brace(s, start_pos):
    """Find the position of the matching closing brace for the opening brace at start_pos.
    Assumes s[start_pos] == '{'.
    Returns the index of the matching '}' or -1 if not found.
    """
    stack = 1
    pos = start_pos + 1
    while pos < len(s):
        ch = s[pos]
        if ch == '{':
            stack += 1
        elif ch == '}':
            stack -= 1
            if stack == 0:
                return pos
        pos += 1
    return -1

# Read the current handler.go
with open('handler.go', 'r', encoding='utf-8') as f:
    content = f.read()

# 1. Update welcomeMenu
new_welcome = """const welcomeMenu = `Bienvenue sur GUV (Guichet unique des voyages) 🚌

1️⃣ Recherche directe de voyages
2️⃣ Recherche par agence
3️⃣ Créer une agence
4️⃣ 🎫 Vérifier mon billet (Check Ticket)
5️⃣ Aide & Assistance

Répondez avec le numéro de votre choix.`"""
pattern = r'const welcomeMenu = `.*?\n\s*`\n'
content = re.sub(pattern, new_welcome + '\n\n', content, flags=re.DOTALL)

# 2. Replace the process function with a modified version from the original
# Read the original file to get the original process function
with open('../../original_handler_original.go', 'r', encoding='utf-8') as f:
    original_content = f.read()

# Find the process function in the original
orig_process_start = original_content.find('func process')
if orig_process_start == -1:
    print("Error: func process not found in original")
    exit(1)
# Find the opening brace of the function body
brace_open = original_content.find('{', orig_process_start)
if brace_open == -1:
    print("Error: opening brace not found in original process function")
    exit(1)
# Find the matching closing brace
brace_close = find_matching_brace(original_content, brace_open)
if brace_close == -1:
    print("Error: matching closing brace not found in original process function")
    exit(1)
# Extract the original process function (including the signature and braces)
original_process = original_content[orig_process_start:brace_close+1]

# Now modify the original_process to have the new StepMenu case and add StepCheckMenu case.
# We'll do string replacements on original_process.

# First, replace the StepMenu case block.
# We'll find the "case StepMenu:" inside original_process.
step_menu_start = original_process.find('case StepMenu:')
if step_menu_start == -1:
    print("Error: case StepMenu: not found in original process function")
    exit(1)
# Split original_process into lines
orig_lines = original_process.split('\n')
# Find the line index of the case StepMenu:
line_index = -1
for i, line in enumerate(orig_lines):
    if line.startswith('case StepMenu:'):
        line_index = i
        break
if line_index == -1:
    print("Error: case StepMenu: not found in orig_lines")
    exit(1)
# Find the next line that starts a new case or default at the same level (no leading whitespace)
i = line_index + 1
while i < len(orig_lines):
    stripped = orig_lines[i].lstrip()
    if stripped.startswith('case ') or stripped.startswith('default:'):
        if orig_lines[i].startswith(stripped):
            break
    i += 1
# Replace lines[line_index:i] with the new StepMenu block
new_step_menu_block = """case StepMenu:
	switch text {
	case "1":
		s.PreviousStep = s.Step
		s.Step = StepSearchOrigin
		return getPromptForStep(s)
	case "2":
		agencies, err := getActiveAgencies()
		if err != nil {
			return "Erreur lors de la récupération des agences. Tapez *menu* pour réessayer."
		}
		log.Printf("Found %d active agencies for phone %s", len(agencies), phone)
		if len(agencies) == 0 {
			return "Aucune agence active trouvée. Tapez *menu* pour réessayer."
		}
		s.Agencies = agencies
		s.PreviousStep = s.Step
		s.Step = StepAgencyChoice
		return getPromptForStep(s)
	case "3":
		s.PreviousStep = s.Step
		s.Step = StepAgencyName
		return getPromptForStep(s)
	case "4":
		s.PreviousStep = s.Step
		s.Step = StepCheckMenu
		return getPromptForStep(s)
	case "5":
		// Aide & Assistance
		return "📞 Pour toute assistance ou information, notre service client est à votre disposition :\n- Ligne directe & WhatsApp : +243974645174\n- Email : ayaletrans@gmail.com"
	default:
		return welcomeMenu
}"""
orig_lines[line_index:i] = [new_step_menu_block]
# Now, add the StepCheckMenu case after the StepMenu case block
# Re-split the updated orig_lines
orig_lines = '\n'.join(orig_lines).split('\n')
line_index = -1
for i, line in enumerate(orig_lines):
    if line.startswith('case StepMenu:'):
        line_index = i
        break
if line_index == -1:
    print("Error: case StepMenu: not found after update")
    exit(1)
# Find the end of the StepMenu case block we just inserted
i = line_index + 1
while i < len(orig_lines):
    stripped = orig_lines[i].lstrip()
    if stripped.startswith('case ') or stripped.startswith('default:'):
        if orig_lines[i].startswith(stripped):
            break
    i += 1
# Build the StepCheckMenu case block
new_step_check_menu_block = """case StepCheckMenu:
	switch text {
	case "1":
		s.PreviousStep = s.Step
		s.Step = StepPassengerCheckType
		return getPromptForStep(s)
	case "2":
		s.PreviousStep = s.Step
		s.Step = StepAgentPinInput
		return getPromptForStep(s)
	default:
		return getPromptForStep(s)
}"""
orig_lines.insert(i, new_step_check_menu_block)
# Join back
modified_process = '\n'.join(orig_lines)

# Replace the process function in the current content with the modified process
curr_process_start = content.find('func process')
if curr_process_start == -1:
    print("Error: func process not found in current content")
    exit(1)
brace_open_curr = content.find('{', curr_process_start)
if brace_open_curr == -1:
    print("Error: opening brace not found in current process function")
    exit(1)
brace_close_curr = find_matching_brace(content, brace_open_curr)
if brace_close_curr == -1:
    print("Error: matching closing brace not found in current process function")
    exit(1)
content = content[:curr_process_start] + modified_process + content[brace_close_curr+1:]

# Write the file back
with open('handler.go', 'w', encoding='utf-8') as f:
    f.write(content)

print("handler.go updated successfully")
