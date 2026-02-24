document.addEventListener('DOMContentLoaded', () => {
    const cypherQueryEl = document.getElementById('cypherQuery');
    const runQueryBtn = document.getElementById('runQueryBtn');
    const queriesFileEl = document.getElementById('queriesFile');
    const uploadBtn = document.getElementById('uploadBtn');
    const getMasterInfoBtn = document.getElementById('getMasterInfoBtn');
    const masterInfoEl = document.getElementById('masterInfo');
    const getWorkerInfoBtn = document.getElementById('getWorkerInfoBtn');
    const workerInfoEl = document.getElementById('workerInfo');
    const workerSelectEl = document.getElementById('workerSelect');
    const responseEl = document.getElementById('response');
    const graphContainer = document.getElementById('graph-container');

    let network = null;

    function displayMessage(el, message, isError = false) {
        el.textContent = message;
        el.style.color = isError ? 'red' : 'green';
    }

    function renderGraph(graphData) {
        if (network !== null) {
            network.destroy();
            network = null;
        }

        if (!graphData || (!graphData.nodes && !graphData.edges)) {
            graphContainer.innerHTML = '<span>Нет данных для визуализации</span>';
            return;
        }

        const nodes = new vis.DataSet(graphData.nodes.map(node => {
            let title = `<div><b>${node.label} (ID: ${node.id})</b><hr style="margin: 4px 0;">`;
            if (node.props) {
                for (const [key, value] of Object.entries(node.props)) {
                    title += `<b>${key}:</b> ${JSON.stringify(value)}<br>`;
                }
            }
            title += '</div>';
            return {
                id: node.id,
                label: `${node.label}\n(ID: ${node.id})`,
                title: title
            };
        }));

        // **MODIFIED**: Now creates a detailed tooltip for edges as well.
        const edges = new vis.DataSet(graphData.edges.map(edge => {
            const edgeLabel = edge.label || '';
            let title = `<div><b>${edgeLabel}</b><hr style="margin: 4px 0;">`;
            if (edge.props && Object.keys(edge.props).length > 0) {
                 for (const [key, value] of Object.entries(edge.props)) {
                    title += `<b>${key}:</b> ${JSON.stringify(value)}<br>`;
                }
            } else {
                title += 'No properties';
            }
            title += '</div>';

            return {
                from: edge.from,
                to: edge.to,
                label: edgeLabel,
                title: title, // Add the tooltip to the edge
                arrows: 'to'
            };
        }));

        const data = { nodes: nodes, edges: edges };

        const options = {
            nodes: {
                shape: 'box',
                margin: 10,
                font: { multi: 'html', size: 14 },
                widthConstraint: { maximum: 250 }
            },
            edges: {
                width: 2,
                font: { align: 'top', size: 12, color: '#555' },
                smooth: {
                    type: 'continuous'
                }
            },
            physics: {
                solver: 'forceAtlas2Based',
            },
            interaction: {
                hover: true,
                tooltipDelay: 200
            }
        };

        network = new vis.Network(graphContainer, data, options);
    }

    runQueryBtn.addEventListener('click', () => {
        const query = cypherQueryEl.value.trim();
        if (!query) return;

        responseEl.textContent = 'Processing...';
        responseEl.style.color = 'black';
        graphContainer.innerHTML = '<span style="color: #888;">Загрузка...</span>';

        fetch('/api/cypher', {
            method: 'POST',
            body: query,
            headers: { 'Content-Type': 'text/plain' }
        })
        .then(response => {
            if (!response.ok) {
                return response.text().then(text => { throw new Error(text || `Server error with status ${response.status}`) });
            }
            return response.json();
        })
        .then(data => {
            if (data.nodes || data.edges) {
                displayMessage(responseEl, 'Query successful. Rendering graph...');
                renderGraph(data);
            } else if (data.status) {
                 displayMessage(responseEl, `Query executed successfully. Status: ${data.status}`);
            }
        })
        .catch(err => {
            displayMessage(responseEl, `Error: ${err.message}`, true);
        });
    });

    uploadBtn.addEventListener('click', () => {
        const file = queriesFileEl.files[0];
        if (!file) return;

        const formData = new FormData();
        formData.append('queriesFile', file);
        responseEl.textContent = 'Uploading and processing...';
        responseEl.style.color = 'black';

        fetch('/api/upload', {
            method: 'POST',
            body: formData
        })
        .then(response => {
            if (!response.ok) {
                return response.text().then(text => { throw new Error(text) });
            }
            return response.json();
        })
        .then(data => {
            const message = `Success! ${data.message} Queries processed: ${data.queries_processed}`;
            displayMessage(responseEl, message);
        })
        .catch(err => {
            displayMessage(responseEl, `Error: ${err.message}`, true);
        });
    });

    getMasterInfoBtn.addEventListener('click', () => {
        masterInfoEl.textContent = 'Fetching master info...';
        fetch('/api/info')
            .then(res => res.json())
            .then(data => {
                masterInfoEl.textContent = JSON.stringify(data, null, 2);
                workerSelectEl.innerHTML = '';
                if (data.WorkerAddrs) {
                    Object.keys(data.WorkerAddrs).forEach(id => {
                        const option = document.createElement('option');
                        option.value = id;
                        option.textContent = `Worker ${id} (${data.WorkerAddrs[id]})`;
                        workerSelectEl.appendChild(option);
                    });
                }
            })
            .catch(err => {
                masterInfoEl.textContent = `Error: ${err.message}`;
                masterInfoEl.style.color = 'red';
            });
    });

    getWorkerInfoBtn.addEventListener('click', () => {
        const workerId = workerSelectEl.value;
        if (workerId === '') {
            workerInfoEl.textContent = 'Please select a worker first.';
            return;
        }
        workerInfoEl.textContent = 'Fetching worker info...';
        fetch(`/api/worker-info?id=${workerId}`)
            .then(res => res.json())
            .then(data => {
                workerInfoEl.textContent = JSON.stringify(data, null, 2);
            })
            .catch(err => {
                workerInfoEl.textContent = `Error: ${err.message}`;
                workerInfoEl.style.color = 'red';
            });
    });

    getMasterInfoBtn.click();
});