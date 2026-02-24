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
    const elementDetailsEl = document.getElementById('elementDetails');
    const fileNameDisplay = document.getElementById('file-name');


    queriesFileEl.addEventListener('change', function () {
        if (this.files.length > 0) {
            fileNameDisplay.textContent = this.files[0].name;
        } else {
            fileNameDisplay.textContent = 'Choose File';
        }
    });


    let network = null;
    let graphNodes = new vis.DataSet();
    let graphEdges = new vis.DataSet();


    function displayMessage(el, message, isError = false) {
        el.textContent = message;
        el.style.color = isError ? 'red' : 'green';
    }


    function renderStructuredInfo(element, data) {
        let html = '';
        if (data) {
            
            if (data.WorkerAddrs && data.PartitionInfoMap) {
                
                html = formatMasterInfo(data);
            } else if (data.vertices !== undefined || data.Vertices !== undefined) {
                
                html = formatWorkerInfo(data);
            } else {
                
                html = `<div class="info-display">${buildHtml(data)}</div>`;
            }
        } else {
            html = '<p>No data available.</p>';
        }
        element.innerHTML = html;
    
        function formatMasterInfo(data) {
            let html = '<div class="info-display">';
            
            
            if (data.PartitionInfoMap && typeof data.PartitionInfoMap === 'object') {
                const partitionKeys = Object.keys(data.PartitionInfoMap).sort((a, b) => parseInt(a) - parseInt(b));
                
                partitionKeys.forEach(partitionId => {
                    const partition = data.PartitionInfoMap[partitionId];
                    if (!partition) return; 
                    
                    html += `<div class="metadata-section">`;
                    html += `<div class="metadata-title">Фрагмент ${partitionId}:</div>`;
                    
                    
                    const vertices = partition.Vertices || partition.vertices || [];
                    const verticesArray = Array.isArray(vertices) ? vertices : [];
                    html += `<div class="metadata-list">`;
                    html += `<div class="metadata-item"><strong>Вершины:</strong> ${verticesArray.length > 0 ? verticesArray.join(', ') : 'нет'}</div>`;
                    
                    
                    html += `<div class="metadata-item"><strong>Ребра:</strong></div>`;
                    const edges = partition.Edges || partition.edges || {};
                    const edgesObj = (typeof edges === 'object' && edges !== null) ? edges : {};
                    
                    if (Object.keys(edgesObj).length > 0) {
                        html += `<div style="margin-left: 15px;">`;
                        Object.keys(edgesObj).sort((a, b) => parseInt(a) - parseInt(b)).forEach(from => {
                            const toList = edgesObj[from];
                            const toArray = Array.isArray(toList) ? toList : [];
                            html += `<div class="metadata-item">${from} -> [${toArray.join(', ')}]</div>`;
                        });
                        html += `</div>`;
                    } else {
                        html += `<div style="margin-left: 15px;"><div class="metadata-item">нет</div></div>`;
                    }
                    
                    
                    const ghostVertices = partition.GhostVertices || partition.ghost_vertices || [];
                    const ghostArray = Array.isArray(ghostVertices) ? ghostVertices : [];
                    html += `<div class="metadata-item"><strong>Пограничные вершины:</strong> ${ghostArray.length > 0 ? ghostArray.join(', ') : 'нет'}</div>`;
                    
                    html += `</div>`;
                    html += `</div>`;
                });
            }
            
            
            html += `<div class="metadata-section">`;
            html += `<div class="metadata-title">Общая информация:</div>`;
            html += `<div class="metadata-list">`;
            
            
            let totalVertices = 0;
            let totalEdges = 0;
            if (data.PartitionInfoMap && typeof data.PartitionInfoMap === 'object') {
                Object.values(data.PartitionInfoMap).forEach(partition => {
                    if (!partition) return; 
                    
                    const vertices = partition.Vertices || partition.vertices || [];
                    const verticesArray = Array.isArray(vertices) ? vertices : [];
                    totalVertices += verticesArray.length;
                    
                    const edges = partition.Edges || partition.edges || {};
                    const edgesObj = (typeof edges === 'object' && edges !== null) ? edges : {};
                    Object.values(edgesObj).forEach(edgeList => {
                        const edgeArray = Array.isArray(edgeList) ? edgeList : [];
                        totalEdges += edgeArray.length;
                    });
                });
            }
            
            html += `<div class="metadata-item"><strong>Число вершин:</strong> ${totalVertices}</div>`;
            html += `<div class="metadata-item"><strong>Число ребер:</strong> ${totalEdges}</div>`;
            html += `<div class="metadata-item"><strong>Число подчиненных узлов:</strong> ${data.NumPartitions || 0}</div>`;
            
            
            if (data.WorkerAddrs && typeof data.WorkerAddrs === 'object') {
                html += `<div class="metadata-item"><strong>Адреса узлов:</strong></div>`;
                html += `<div style="margin-left: 15px;">`;
                Object.keys(data.WorkerAddrs).sort((a, b) => parseInt(a) - parseInt(b)).forEach(id => {
                    const addr = data.WorkerAddrs[id];
                    html += `<div class="metadata-item">Узел ${id}: ${addr || 'не указан'}</div>`;
                });
                html += `</div>`;
            }
            
            
            const avgLoad = calculateAverageLoadFromData(data);
            html += `<div class="metadata-item"><strong>Средняя нагрузка:</strong> ${avgLoad.toFixed(2)}</div>`;
            
            html += `</div>`;
            html += `</div>`;
            html += `</div>`;
            
            return html;
        }
    
        function formatWorkerInfo(data) {
            let html = '<div class="info-display">';
            
            
            const vertices = data.Vertices || data.vertices || [];
            const verticesArray = Array.isArray(vertices) ? vertices : [];
            html += `<div class="metadata-item"><strong>Вершины:</strong> ${verticesArray.length > 0 ? verticesArray.join(', ') : 'нет'}</div>`;
            
            
            html += `<div class="metadata-item"><strong>Ребра:</strong></div>`;
            const edges = data.Edges || data.edges || {};
            const edgesObj = (typeof edges === 'object' && edges !== null) ? edges : {};
            
            if (Object.keys(edgesObj).length > 0) {
                html += `<div style="margin-left: 15px;">`;
                Object.keys(edgesObj).sort((a, b) => parseInt(a) - parseInt(b)).forEach(from => {
                    const toList = edgesObj[from];
                    const toArray = Array.isArray(toList) ? toList : [];
                    html += `<div class="metadata-item">${from} -> [${toArray.join(', ')}]</div>`;
                });
                html += `</div>`;
            } else {
                html += `<div style="margin-left: 15px;"><div class="metadata-item">нет</div></div>`;
            }
            
            
            const ghostVertices = data.GhostVertices || data.ghost_vertices || [];
            const ghostArray = Array.isArray(ghostVertices) ? ghostVertices : [];
            html += `<div class="metadata-item"><strong>Пограничные вершины:</strong> ${ghostArray.length > 0 ? ghostArray.join(', ') : 'нет'}</div>`;
            
            html += `</div>`;
            return html;
        }
        function calculateAverageLoadFromData(data) {
            let totalEdges = 0;
            let partitionCount = 0;
            
            if (data.PartitionInfoMap && typeof data.PartitionInfoMap === 'object') {
                Object.values(data.PartitionInfoMap).forEach(partition => {
                    if (!partition) return; 
                    
                    const edges = partition.Edges || partition.edges || {};
                    const edgesObj = (typeof edges === 'object' && edges !== null) ? edges : {};
                    Object.values(edgesObj).forEach(edgeList => {
                        const edgeArray = Array.isArray(edgeList) ? edgeList : [];
                        totalEdges += edgeArray.length;
                    });
                    partitionCount++;
                });
            }
            
            return partitionCount > 0 ? totalEdges / partitionCount : 0;
        }
        function buildHtml(obj, indent = 0) {
            let currentHtml = '';
            for (const key in obj) {
                if (Object.prototype.hasOwnProperty.call(obj, key)) {
                    const value = obj[key];
                    const keyClass = 'info-key';
                    const valueClass = 'info-value';
    
                    if (typeof value === 'object' && value !== null && !Array.isArray(value)) {
                        currentHtml += `<div class="info-row"><span class="${keyClass}">${key}:</span></div>`;
                        currentHtml += `<div class="info-nested">${buildHtml(value, indent + 1)}</div>`;
                    } else if (Array.isArray(value)) {
                        currentHtml += `<div class="info-row"><span class="${keyClass}">${key}:</span></div>`;
                        currentHtml += `<div class="info-nested">`;
                        value.forEach((item, index) => {
                            if (typeof item === 'object' && item !== null) {
                                currentHtml += `<div class="info-row"><span class="${valueClass}">[${index}]:</span></div>`;
                                currentHtml += `<div class="info-nested">${buildHtml(item, indent + 2)}</div>`;
                            } else {
                                if (key === 'Vertices' || key === 'vertices' || key === 'internal_vertices' || key === 'extended_vertices' || key === 'GhostVertices') {
                                    currentHtml += `<div class="info-row"><span class="${valueClass}">${JSON.stringify(item)}</span></div>`;
                                } else if (key === 'Edges' || key === 'edges' || key === 'CrossingEdges') {
                                    if (item.V1 !== undefined && item.V2 !== undefined) {
                                        currentHtml += `<div class="info-row"><span class="${valueClass}">${item.V1} → ${item.V2}</span></div>`;
                                        if (item.Labels && Object.keys(item.Labels).length > 0) {
                                            currentHtml += `<div class="info-nested">${buildHtml(item.Labels, indent + 3)}</div>`;
                                        }
                                    } else {
                                        currentHtml += `<div class="info-row"><span class="${valueClass}">${JSON.stringify(item)}</span></div>`;
                                    }
                                } else {
                                    currentHtml += `<div class="info-row"><span class="${valueClass}">${JSON.stringify(item)}</span></div>`;
                                }
                            }
                        });
                        currentHtml += `</div>`;
                    } else {
                        currentHtml += `<div class="info-row"><span class="${keyClass}">${key}:</span> <span class="${valueClass}">${JSON.stringify(value)}</span></div>`;
                    }
                }
            }
            return currentHtml;
        }
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


        graphNodes.clear();
        graphEdges.clear();


        graphNodes.add(graphData.nodes.map(node => {
            return {
                id: node.id,
                label: `${node.label}\n(ID: ${node.id})`,
                raw: node
            };
        }));


        graphEdges.add(graphData.edges.map(edge => {
            const edgeLabel = edge.label || '';
            return {
                from: edge.from,
                to: edge.to,
                label: edgeLabel,
                arrows: 'to',
                raw: edge
            };
        }));

        const data = { nodes: graphNodes, edges: graphEdges };

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
                stabilization: {
                    iterations: 2000,
                    updateInterval: 100,
                    fit: true
                }
            },
            interaction: {
                hover: true,
                tooltipDelay: 0,
                selectConnectedEdges: false
            }
        };

        network = new vis.Network(graphContainer, data, options);


        network.on("hoverNode", function (params) {
            const nodeId = params.node;
            const node = graphNodes.get(nodeId);
            if (node && node.raw) {
                renderStructuredInfo(elementDetailsEl, node.raw);
            }
        });

        network.on("blurNode", function (params) {
            elementDetailsEl.innerHTML = '<p>Hover over a node or edge to see its details here.</p>';
        });

        network.on("hoverEdge", function (params) {
            const edgeId = params.edge;
            const edge = graphEdges.get(edgeId);
            if (edge && edge.raw) {
                renderStructuredInfo(elementDetailsEl, edge.raw);
            }
        });

        network.on("blurEdge", function (params) {
            elementDetailsEl.innerHTML = '<p>Hover over a node or edge to see its details here.</p>';
        });



        network.on("click", function (params) {
            if (params.nodes.length > 0) {
                const nodeId = params.nodes[0];
                const node = graphNodes.get(nodeId);
                if (node && node.raw) {
                    renderStructuredInfo(elementDetailsEl, node.raw);
                }
            } else if (params.edges.length > 0) {
                const edgeId = params.edges[0];
                const edge = graphEdges.get(edgeId);
                if (edge && edge.raw) {
                    renderStructuredInfo(elementDetailsEl, edge.raw);
                }
            }
            else {
                elementDetailsEl.innerHTML = '<p>Hover over a node or edge to see its details here.</p>';
            }
        });
    }

    runQueryBtn.addEventListener('click', () => {
        const query = cypherQueryEl.value.trim();
        if (!query) return;

        responseEl.textContent = 'Processing...';
        responseEl.style.color = 'black';
        graphContainer.innerHTML = '<span style="color: #888;">Загрузка...</span>';
        elementDetailsEl.innerHTML = '<p>Hover over a node or edge to see its details here.</p>';

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
        elementDetailsEl.innerHTML = '<p>Hover over a node or edge to see its details here.</p>';

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
                renderStructuredInfo(masterInfoEl, data);
                workerSelectEl.innerHTML = '';
                if (data.WorkerAddrs) {
                    Object.keys(data.WorkerAddrs).forEach(id => {
                        const option = document.createElement('option');
                        option.value = id;
                        option.textContent = `Узел ${id} (${data.WorkerAddrs[id]})`;
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
                renderStructuredInfo(workerInfoEl, data);
            })
            .catch(err => {
                workerInfoEl.textContent = `Error: ${err.message}`;
                workerInfoEl.style.color = 'red';
            });
    });

    getMasterInfoBtn.click();
});