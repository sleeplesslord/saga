(() => {
  'use strict';

  const escapeHTML = value => String(value ?? '').replace(/[&<>'"]/g, character => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    "'": '&#39;',
    '"': '&quot;'
  })[character]);

  const safeURL = value => {
    const url = String(value ?? '').trim();
    if (/^(https?:|mailto:)/i.test(url)) return url;
    if (/^[a-z][a-z\d+.-]*:/i.test(url) || /^[\\/]{2}/.test(url)) return '';
    return url;
  };

  function inline(source) {
    const tokens = [];
    const hold = html => {
      const token = `\u0000${tokens.length}\u0000`;
      tokens.push(html);
      return token;
    };

    let text = String(source ?? '').replace(/\u0000/g, '\ufffd');
    text = text.replace(/(`+)([\s\S]*?)\1/g, (_, ticks, code) => hold(`<code>${escapeHTML(code.trim())}</code>`));
    text = text.replace(/\[([^\]\n]+)\]\(([^\s)]+)(?:\s+["']([^"']*)["'])?\)/g, (match, label, destination, title) => {
      const url = safeURL(destination);
      if (!url) return label;
      const external = /^https?:/i.test(url);
      const titleAttribute = title ? ` title="${escapeHTML(title)}"` : '';
      const externalAttributes = external ? ' target="_blank" rel="noopener noreferrer"' : '';
      return hold(`<a href="${escapeHTML(url)}"${titleAttribute}${externalAttributes}>${inline(label)}</a>`);
    });
    text = text.replace(/<((?:https?:\/\/|mailto:)[^>\s]+)>/gi, (_, destination) => {
      const url = safeURL(destination);
      const external = /^https?:/i.test(url);
      const externalAttributes = external ? ' target="_blank" rel="noopener noreferrer"' : '';
      return hold(`<a href="${escapeHTML(url)}"${externalAttributes}>${escapeHTML(destination)}</a>`);
    });

    text = escapeHTML(text)
      .replace(/~~([^~\n]+)~~/g, '<del>$1</del>')
      .replace(/\*\*([^*\n]+)\*\*/g, '<strong>$1</strong>')
      .replace(/__([^_\n]+)__/g, '<strong>$1</strong>')
      .replace(/(^|[^*])\*([^*\n]+)\*(?!\*)/g, '$1<em>$2</em>')
      .replace(/(^|[^\w])_([^_\n]+)_(?!\w)/g, '$1<em>$2</em>')
      .replace(/ {2,}\n/g, '<br>')
      .replace(/\n/g, ' ');

    return text.replace(/\u0000(\d+)\u0000/g, (_, index) => tokens[Number(index)]);
  }

  const splitTableRow = line => {
    let value = line.trim();
    if (value.startsWith('|')) value = value.slice(1);
    if (value.endsWith('|')) value = value.slice(0, -1);
    return value.split(/(?<!\\)\|/).map(cell => cell.trim().replace(/\\\|/g, '|'));
  };

  const tableDivider = line => {
    if (!line.includes('|')) return null;
    const cells = splitTableRow(line);
    return cells.length && cells.every(cell => /^:?-{3,}:?$/.test(cell)) ? cells : null;
  };

  const startsBlock = (lines, index) => {
    const line = lines[index] ?? '';
    const next = lines[index + 1] ?? '';
    return /^ {0,3}(#{1,6})\s+/.test(line) ||
      /^ {0,3}(`{3,}|~{3,})/.test(line) ||
      /^ {0,3}>/.test(line) ||
      /^ {0,3}(?:[-+*]|\d+[.)])\s+/.test(line) ||
      /^ {0,3}(?:-{3,}|\*{3,}|_{3,})\s*$/.test(line) ||
      (/\|/.test(line) && Boolean(tableDivider(next))) ||
      /^(?: {4}|\t)/.test(line);
  };

  function renderBlocks(source, depth = 0) {
    const lines = String(source ?? '').replace(/\r\n?/g, '\n').split('\n');
    const output = [];
    let index = 0;

    while (index < lines.length) {
      const line = lines[index];
      if (!line.trim()) {
        index += 1;
        continue;
      }

      const fence = line.match(/^ {0,3}(`{3,}|~{3,})\s*([^\s]*)\s*$/);
      if (fence) {
        const marker = fence[1][0];
        const minimum = fence[1].length;
        const code = [];
        index += 1;
        while (index < lines.length && !new RegExp(`^ {0,3}${marker}{${minimum},}\\s*$`).test(lines[index])) {
          code.push(lines[index]);
          index += 1;
        }
        if (index < lines.length) index += 1;
        const language = fence[2] ? `<span class="code-language">${escapeHTML(fence[2])}</span>` : '';
        output.push(`<div class="code-block">${language}<pre><code>${escapeHTML(code.join('\n'))}</code></pre></div>`);
        continue;
      }

      if (/^(?: {4}|\t)/.test(line)) {
        const code = [];
        while (index < lines.length && (/^(?: {4}|\t)/.test(lines[index]) || !lines[index].trim())) {
          code.push(lines[index].replace(/^(?: {4}|\t)/, ''));
          index += 1;
        }
        while (code.length && !code[code.length - 1]) code.pop();
        output.push(`<div class="code-block"><pre><code>${escapeHTML(code.join('\n'))}</code></pre></div>`);
        continue;
      }

      const heading = line.match(/^ {0,3}(#{1,6})\s+(.+?)\s*#*\s*$/);
      if (heading) {
        const level = Math.min(heading[1].length + 3, 6);
        output.push(`<h${level}>${inline(heading[2])}</h${level}>`);
        index += 1;
        continue;
      }

      if (/^ {0,3}(?:-{3,}|\*{3,}|_{3,})\s*$/.test(line)) {
        output.push('<hr>');
        index += 1;
        continue;
      }

      if (/^ {0,3}>/.test(line)) {
        const quote = [];
        while (index < lines.length && (/^ {0,3}>/.test(lines[index]) || !lines[index].trim())) {
          quote.push(lines[index].replace(/^ {0,3}> ?/, ''));
          index += 1;
        }
        const contents = depth < 6 ? renderBlocks(quote.join('\n'), depth + 1) : `<p>${inline(quote.join('\n'))}</p>`;
        output.push(`<blockquote>${contents}</blockquote>`);
        continue;
      }

      const listItem = line.match(/^ {0,3}([-+*]|\d+[.)])\s+(.+)$/);
      if (listItem) {
        const ordered = /^\d/.test(listItem[1]);
        const items = [];
        while (index < lines.length) {
          const match = lines[index].match(/^ {0,3}([-+*]|\d+[.)])\s+(.+)$/);
          if (!match || /^\d/.test(match[1]) !== ordered) break;
          let contents = match[2];
          const continuation = [];
          index += 1;
          while (index < lines.length && /^ {2,}\S/.test(lines[index]) && !/^ {0,3}(?:[-+*]|\d+[.)])\s+/.test(lines[index])) {
            continuation.push(lines[index].trim());
            index += 1;
          }
          if (continuation.length) contents += `\n${continuation.join('\n')}`;
          const task = contents.match(/^\[([ xX])\]\s+(.+)$/s);
          if (task) {
            const checked = task[1].toLowerCase() === 'x';
            items.push(`<li class="task-item"><input class="task-check" type="checkbox" disabled aria-label="${checked ? 'Completed' : 'Not completed'}"${checked ? ' checked' : ''}><span>${inline(task[2])}</span></li>`);
          } else {
            items.push(`<li>${inline(contents)}</li>`);
          }
        }
        const tag = ordered ? 'ol' : 'ul';
        output.push(`<${tag}>${items.join('')}</${tag}>`);
        continue;
      }

      if (line.includes('|') && tableDivider(lines[index + 1] ?? '')) {
        const headings = splitTableRow(line);
        const dividers = splitTableRow(lines[index + 1]);
        const alignments = dividers.map(cell => cell.startsWith(':') && cell.endsWith(':') ? 'center' : cell.endsWith(':') ? 'right' : 'left');
        const rows = [];
        index += 2;
        while (index < lines.length && lines[index].includes('|') && lines[index].trim()) {
          rows.push(splitTableRow(lines[index]));
          index += 1;
        }
        const header = headings.map((cell, cellIndex) => `<th class="align-${alignments[cellIndex] || 'left'}">${inline(cell)}</th>`).join('');
        const body = rows.map(row => `<tr>${headings.map((_, cellIndex) => `<td class="align-${alignments[cellIndex] || 'left'}">${inline(row[cellIndex] || '')}</td>`).join('')}</tr>`).join('');
        output.push(`<div class="markdown-table"><table><thead><tr>${header}</tr></thead><tbody>${body}</tbody></table></div>`);
        continue;
      }

      const paragraph = [line.trim()];
      index += 1;
      while (index < lines.length && lines[index].trim() && !startsBlock(lines, index)) {
        paragraph.push(lines[index].trim());
        index += 1;
      }
      output.push(`<p>${inline(paragraph.join('\n'))}</p>`);
    }

    return output.join('');
  }

  globalThis.SagaMarkdown = Object.freeze({ render: source => renderBlocks(source) });
})();
