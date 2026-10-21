import sys

def main():
    try:
        with open('internal/whatsapp/handler.go', 'r', encoding='utf-8') as f:
            content = f.read()
    except Exception as e:
        print(f'Error reading file: {e}')
        return 1

    # Exact old string as determined from inspection
    old_str = '\tcase StepCheckMenu:\n\t\treturn "🔍 Espace Vérification & Guichet :\\n\\n1️⃣ Voir l\\'état d\\'avancement\\n2️⃣ Guichet (Réservé aux agents 🧑‍💼)\\n\\nRépondez avec le numéro de votre choix."'

    # New string with switch
    new_str = '''\\tswitch text {
\\tcase \\"1\\":\\n\\t\\ts.PreviousStep = s.Step\\n\\t\\ts.Step = StepPassengerCheckType\\n\\t\\treturn getPromptForStep(s)\\n\\tcase \\"2\\":\\n\\t\\ts.PreviousStep = s.Step\\n\\t\\ts.Step = StepAgentPinInput\\n\\t\\treturn getPromptForStep(s)\\n\\tdefault:\\n\\t\\treturn \\"🔍 Espace Vérification & Guichet :\\\\n\\\\n1️⃣ Voir l\\'état d\\'avancement\\\\n\\\\n2️⃣ Guichet (Réservé aux agents 🧑‍💼)\\\\\\n\\\\nRépondez avec le numéro de votre choix.\\"\n\\t}'''

    # Let's build the new string more carefully
    new_str_lines = [
        '\tswitch text {',
        '\tcase "1":',
        '\t\ts.PreviousStep = s.Step',
        '\t\ts.Step = StepPassengerCheckType',
        '\t\treturn getPromptForStep(s)',
        '\tcase "2":',
        '\t\ts.PreviousStep = s.Step',
        '\t\ts.Step = StepAgentPinInput',
        '\t\treturn getPromptForStep(s)',
        '\tdefault:',
        '\t\treturn "🔍 Espace Vérification & Guichet :\\n\\n1️⃣ Voir l\\'état d\\'avancement\\n2️⃣ Guichet (Réservé aux agents 🧑‍💼)\\n\\nRépondez avec le numéro de votre choix."',
        '\t}'
    ]
    new_str = '\\n'.join(new_str_lines)

    if old_str in content:
        content = content.replace(old_str, new_str)
        try:
            with open('internal/whatsapp/handler.go', 'w', encoding='utf-8') as f:
                f.write(content)
            print('SUCCESS: Replacement completed')
            return 0
        except Exception as e:
            print(f'Error writing file: {e}')
            return 1
    else:
        print('FAILURE: Old string not found')
        # Debug: show what we have
        print('Looking for case StepCheckMenu...')
        if 'case StepCheckMenu:' in content:
            print('Case line found')
            # Find the line and show context
            lines = content.split('\n')
            for i, line in enumerate(lines):
                if 'case StepCheckMenu:' in line:
                    print(f'Context around line {i+1}:')
                    for j in range(max(0, i-2), min(len(lines), i+5)):
                        marker = '>>> ' if j == i else '    '
                        print(f'{marker}{j+1:4}: {lines[j]}')
                    break
        else:
            print('Case line not found')
        return 1

if __name__ == '__main__':
    sys.exit(main())