/**
 * OpenLayers 地图辅助工具 - 高性能节点渲染
 * 支持万级节点的流畅渲染和交互
 */

class OLTopoHelper {
    constructor() {
        this.map = null;
        this.vectorSource = null;
        this.vectorLayer = null;
        this.lineSource = null;
        this.lineLayer = null;
        this.overlay = null;
        this.selectedFeature = null;
        this.clusterMap = new Map(); // 存储重叠节点分组：key=经纬度, value=节点数组
        this.highlightLayer = null; // 高亮图层
        this.expandedCluster = null; // 当前展开的cluster信息 {locKey, expandLayer}
        this.expandedFeatures = []; // 展开后的临时features
        this.keepExpandedOnViewChange = false; // 视图变化时是否保持展开状态
        this.isAutoLocating = false; // 是否正在自动定位中（防止缩放时收起cluster）
        this.isLocatingAndHighlighting = false; // 是否正在执行定位和高亮操作（防止重复调用）
        this.allNodesCache = []; // 缓存所有节点数据，用于更新连线
        
        // 电子围栏相关
        this.fenceSource = null; // 围栏数据源
        this.fenceLayer = null; // 围栏图层
        this.currentDrawInteraction = null; // 当前绘制交互
        this.fenceModifyInteraction = null; // 围栏修改交互
        this.fenceClickListener = null; // 围栏点击监听器引用
        
        // 节点样式映射
        this.styleConfig = {
            enb: {
                online: { color: '#40BC33', size: 12 },
                offline: { color: '#E88282', size: 12 },
                inactive: { color: '#E88282', size: 12 },
                registed: { color: '#F99C9C', size: 12 },
                granted: { color: '#F8D675', size: 12 },
                authed: { color: '#88D46D', size: 12 }
            },
            cpe: {
                online: { color: '#40BC33', size: 12 },
                offline: { color: '#E88282', size: 12 },
                inactive: { color: '#E88282', size: 12 },
                registed: { color: '#F99C9C', size: 12 },
                granted: { color: '#F8D675', size: 12 },
                authed: { color: '#88D46D', size: 12 }
            },
            gnb: {
                online: { color: '#40BC33', size: 12 },
                offline: { color: '#E88282', size: 12 },
                inactive: { color: '#E88282', size: 12 },
                registed: { color: '#F99C9C', size: 12 },
                granted: { color: '#F8D675', size: 12 },
                authed: { color: '#88D46D', size: 12 }
            },
            gsm: {
                online: { color: '#40BC33', size: 12 },
                offline: { color: '#E88282', size: 12 },
                inactive: { color: '#E88282', size: 12 },
                registed: { color: '#F99C9C', size: 12 },
                granted: { color: '#F8D675', size: 12 },
                authed: { color: '#88D46D', size: 12 }
            }
        };
        
        // SVG模板缓存
        this.svgTemplates = {};
        // 原始SVG内容缓存
        this.originalSvgCache = {};
        // SVG加载Promise缓存
        this.svgLoadingPromises = {};
        // MDT SVG缓存
        this.mdtSvgCache = null;
        // 防抖刷新定时器
        this.refreshTimer = null;
        // 待加载的SVG数量
        this.pendingSvgLoads = 0;
        
        // 测距功能相关变量初始化
        this.measureActive = false;
        this.measureCoordinates = [];
        this.measureSource = null;
        this.measureLayer = null;
        this.measureTempFeature = null;
        this.measureTooltips = [];
        this.measureTempTooltip = null;
        this.measureClickHandler = null;
        this.measureMoveHandler = null;
        this.measureDblClickHandler = null;
        
        // Site Map 测距功能相关变量初始化
        this.measureSiteActive = false;
        this.measureSiteCoordinates = [];
        this.measureSiteSource = null;
        this.measureSiteLayer = null;
        this.measureSiteTempFeature = null;
        this.measureSiteTooltips = [];
        this.measureSiteTempTooltip = null;
        this.measureSiteClickHandler = null;
        this.measureSiteMoveHandler = null;
        this.measureSiteDblClickHandler = null;
        
        // 节点拖拽功能相关变量
        this.translateInteraction = null;
        this.dragEndCallback = null;
        this.isDragEnabled = false;
        this.isDragging = false; // 标记是否正在拖拽
        this.justDragged = false; // 标记是否刚完成拖拽（用于阻止点击）
    }

    /**
     * 防抖刷新vectorSource，避免频繁触发重新渲染
     * @param {number} delay - 延迟时间（毫秒），默认300ms
     */
    debouncedRefresh(delay = 300) {
        if (this.refreshTimer) {
            clearTimeout(this.refreshTimer);
        }
        this.refreshTimer = setTimeout(() => {
            if (this.vectorSource) {
                this.vectorSource.changed();
            }
            this.refreshTimer = null;
        }, delay);
    }

    /**
     * 加载并处理MDT SVG图标（蓝色）
     */
    async loadMdtSvg() {
        if (this.mdtSvgCache) {
            return this.mdtSvgCache;
        }

        try {
            const response = await fetch('/js/ol/images/icon-MDT.svg');
            let svgText = await response.text();
            
            // 替换SVG中的颜色为蓝色
            svgText = svgText.replace(/fill="[^"]*"/g, 'fill="#1890ff"');
            svgText = svgText.replace(/stroke="[^"]*"/g, 'stroke="#1890ff"');
            
            // 如果没有fill属性，添加蓝色fill
            if (!svgText.includes('fill=')) {
                svgText = svgText.replace(/<path/g, '<path fill="#1890ff"');
                svgText = svgText.replace(/<circle/g, '<circle fill="#1890ff"');
            }
            
            const dataUrl = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(svgText);
            this.mdtSvgCache = dataUrl;
            return dataUrl;
        } catch (error) {
            console.error('Failed to load MDT SVG:', error);
            return null;
        }
    }

    /**
     * 异步加载原始SVG文件
     * @param {string} nodeType - 节点类型
     */
    async loadOriginalSvg(nodeType) {
        // 如果已经缓存，直接返回
        if (this.originalSvgCache[nodeType]) {
            return this.originalSvgCache[nodeType];
        }

        // 如果正在加载，返回已有的Promise
        if (this.svgLoadingPromises[nodeType]) {
            return this.svgLoadingPromises[nodeType];
        }

        // SVG文件路径映射
        const iconMap = {
            enb: '/js/ol/images/icon-topo-enb.svg',
            gsm: '/js/ol/images/icon_tpopo_2G.svg',
            gnb: '/js/ol/images/icon_tpopo_5G.svg',
            cpe: '/js/ol/images/icon-topo-cpe.svg'
        };

        const svgPath = iconMap[nodeType] || iconMap.enb;

        // 创建加载Promise
        this.svgLoadingPromises[nodeType] = fetch(svgPath)
            .then(response => response.text())
            .then(svgText => {
                this.originalSvgCache[nodeType] = svgText;
                delete this.svgLoadingPromises[nodeType];
                return svgText;
            })
            .catch(error => {
                console.error(`Failed to load SVG for ${nodeType}:`, error);
                delete this.svgLoadingPromises[nodeType];
                return null;
            });

        return this.svgLoadingPromises[nodeType];
    }

    /**
     * 创建带颜色的SVG Data URL（基于原始SVG文件）
     * @param {string} nodeType - 节点类型
     * @param {string} color - 颜色值
     */
    createColoredSvgUrl(nodeType, color) {
        const cacheKey = `${nodeType}_${color}`;
        if (this.svgTemplates[cacheKey]) {
            return this.svgTemplates[cacheKey];
        }

        // 如果原始SVG还未加载，返回临时占位符
        if (!this.originalSvgCache[nodeType]) {
            // 启动异步加载（不等待）
            this.pendingSvgLoads++;
            this.loadOriginalSvg(nodeType).then(() => {
                this.pendingSvgLoads--;
                // 如果这是最后一个待加载的SVG，立即触发刷新
                if (this.pendingSvgLoads === 0) {
                    this.debouncedRefresh(100);
                }
            });
            
            // 返回临时的简单圆形SVG
            const tempSvg = `<svg viewBox="0 0 1024 1024" xmlns="http://www.w3.org/2000/svg"><circle cx="512" cy="512" r="400" fill="${color}"/></svg>`;
            return 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(tempSvg);
        }

        // 替换SVG中的所有颜色
        let coloredSvg = this.originalSvgCache[nodeType];
        
        // 替换所有 fill="#333333" 为指定颜色
        coloredSvg = coloredSvg.replace(/fill="#333333"/g, `fill="${color}"`);
        
        // 如果没有明确的 fill 属性，在 path 标签中添加
        if (!coloredSvg.includes('fill=')) {
            coloredSvg = coloredSvg.replace(/<path/g, `<path fill="${color}"`);
        }

        const dataUrl = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(coloredSvg);
        this.svgTemplates[cacheKey] = dataUrl;
        return dataUrl;
    }

    /**
     * 预加载所有SVG文件
     */
    async preloadAllSvgs() {
        const nodeTypes = ['enb', 'gsm', 'gnb', 'cpe'];
        const promises = nodeTypes.map(type => this.loadOriginalSvg(type));
        // 同时预加载MDT图标
        promises.push(this.loadMdtSvg());
        await Promise.all(promises);
    }

    /**
     * 初始化地图
     * @param {Object} options - 地图配置选项
     */
    initMap(options) {
        const { containerId, bounds, offlineMapEnable, mapUrl } = options;
        
        // 计算中心点和缩放级别
        const center = ol.proj.fromLonLat([
            (bounds[1] + bounds[3]) / 2, // 经度中心
            (bounds[0] + bounds[2]) / 2  // 纬度中心
        ]);
        
        // 根据边界计算合适的缩放级别
        const lonDiff = Math.abs(bounds[3] - bounds[1]);
        const latDiff = Math.abs(bounds[2] - bounds[0]);
        const maxDiff = Math.max(lonDiff, latDiff);
        
        let zoom = 10;
        if (maxDiff > 10) zoom = 4;
        else if (maxDiff > 5) zoom = 6;
        else if (maxDiff > 1) zoom = 8;
        else if (maxDiff > 0.1) zoom = 12;
        else zoom = 15;

        // 瓦片图层
        const tileUrl = offlineMapEnable 
            ? mapUrl || window.location.origin + '/map/{z}/{x}/{y}.png'
            : 'https://{a-c}.tile.openstreetmap.org/{z}/{x}/{y}.png';

        const tileLayer = new ol.layer.Tile({
            source: new ol.source.XYZ({
                url: tileUrl,
                maxZoom: offlineMapEnable ? 12 : 18
            })
        });

        // 创建地图
        this.map = new ol.Map({
            target: containerId,
            layers: [tileLayer],
            view: new ol.View({
                center: center,
                zoom: zoom,
                minZoom: 4,
                maxZoom: offlineMapEnable ? 12 : 18
            })
        });

        // 自适应窗口大小
        const resize = () => {
            if (this.map) {
                this.map.updateSize();
            }
        };
        window.addEventListener('resize', resize);

        // 注意：SVG预加载已移至外部调用处（eNBTopo_tab.jsp的initMap方法）
        // 以确保首次加载时SVG已完全加载完成

        return this.map;
    }

    /**
     * 创建高性能节点图层 (使用 WebGL 渲染)
     * @param {Array} nodes - 节点数据数组
     */
    createNodesLayer(nodes) {
        if (!this.map) {
            console.error('Map not initialized. Call initMap() first.');
            return;
        }
        
        const features = [];
        
        // 清空并重建重叠节点映射
        this.clusterMap.clear();
        
        // 第一步：按经纬度分组节点
        const locationMap = new Map();
        nodes.forEach(node => {
            if (!node.lat || !node.lon || isNaN(node.lat) || isNaN(node.lon)) {
                return;
            }
            
            const locKey = `${node.lat},${node.lon}`;
            if (!locationMap.has(locKey)) {
                locationMap.set(locKey, []);
            }
            locationMap.get(locKey).push(node);
        });
        
        // 第二步：为每个位置创建 Feature，并记录重叠信息
        locationMap.forEach((nodesAtLocation, locKey) => {
            // 使用第一个节点作为代表节点
            const representativeNode = nodesAtLocation[0];
            
            // 确定节点类型
            let nodeType = representativeNode.type || 'enb';
            if (representativeNode.type === 'enb' && representativeNode.isGSM) {
                nodeType = 'gsm';
            }

            // 确定节点状态（从全局获取statusForm配置）
            const statusForm = (typeof window !== 'undefined' && window.topoStatusForm) ? window.topoStatusForm : null;
            let nodeStatus = this.getNodeStatus(representativeNode, statusForm);

            const feature = new ol.Feature({
                geometry: new ol.geom.Point(
                    ol.proj.fromLonLat([parseFloat(representativeNode.lon), parseFloat(representativeNode.lat)])
                ),
                nodeData: representativeNode, // 代表节点数据
                allNodes: nodesAtLocation, // 该位置的所有节点
                nodeType: nodeType,
                nodeStatus: nodeStatus,
                isCluster: nodesAtLocation.length > 1, // 是否为重叠节点
                clusterSize: nodesAtLocation.length // 重叠数量
            });

            features.push(feature);
            
            // 记录重叠节点分组
            if (nodesAtLocation.length > 1) {
                this.clusterMap.set(locKey, nodesAtLocation);
            }
        });

        // 如果有展开的 cluster，在重新渲染前强制收起
        if (this.expandedCluster) {
            this.collapseCluster(true);
        }

        // 移除旧的节点图层（如果存在）
        if (this.vectorLayer && this.map) {
            this.map.removeLayer(this.vectorLayer);
            this.vectorLayer = null;
            this.vectorSource = null;
        }

        // 创建矢量数据源
        this.vectorSource = new ol.source.Vector({
            features: features
        });

        // 使用普通 Vector Layer (因为 WebGLPoints 在 v8+ 中需要特殊配置)
        // 对于万级节点，使用优化的渲染策略
        this.vectorLayer = new ol.layer.Vector({
            source: this.vectorSource,
            style: (feature) => this.getFeatureStyle(feature),
            // 启用图层的渲染优化
            declutter: false, // 不自动避让重叠
            renderBuffer: 100, // 渲染缓冲区
            updateWhileAnimating: true,
            updateWhileInteracting: true,
            zIndex: 10 // 确保节点图层在连线上方
        });

        if (this.map) {
            this.map.addLayer(this.vectorLayer);
        }
    }

    /**
     * 创建 CPE 到 ENB 的连线图层
     * @param {Array} cpeNodes - CPE 节点数据（已过滤后的，包含 relaEnb 字段）
     * @param {Array} allNodes - 所有节点数据（用于查找对应的 ENB）
     */
    createLinesLayer(cpeNodes, allNodes) {
        if (!this.map) {
            console.error('Map not initialized. Call initMap() first.');
            return;
        }
        
        // 移除旧的连线图层
        if (this.lineLayer && this.map) {
            this.map.removeLayer(this.lineLayer);
            this.lineLayer = null;
            this.lineSource = null;
        }

        // 创建节点映射表，方便根据 code 快速查找节点
        // 使用多个字段建立映射，因为 relaEnb 可能匹配 code、cellCode 或其他字段
        const nodeMap = new Map();
        const enbNodes = [];
        allNodes.forEach(node => {
            // 使用 code（serial_number）作为主键
            if (node.code) {
                nodeMap.set(node.code, node);
            }
            // 同时使用 cellCode（small_cell_code）作为备用键
            if (node.cellCode) {
                nodeMap.set(node.cellCode, node);
            }
            // 如果有 sn 字段，也添加到映射
            if (node.sn) {
                nodeMap.set(node.sn, node);
            }
            
            if (node.type === 'enb' || node.type === 'gnb' || node.isGSM) {
                enbNodes.push({
                    code: node.code,
                    cellCode: node.cellCode,
                    sn: node.sn
                });
            }
        });
        
        const lineFeatures = [];
        let foundCount = 0;
        let notFoundCount = 0;

        // 遍历 CPE 节点，找到有 relaEnb 的节点并绘制连线
        cpeNodes.forEach((cpeNode, index) => {
            // 检查是否是 CPE 且有 relaEnb
            if (cpeNode.type === 'cpe' && cpeNode.relaEnb) {
                // 查找对应的 ENB 节点
                const enbNode = nodeMap.get(cpeNode.relaEnb);
                
                if (enbNode && enbNode.lat && enbNode.lon && cpeNode.lat && cpeNode.lon) {
                    // 检查ENB节点是否在展开的cluster中
                    let enbCoordinate = ol.proj.fromLonLat([parseFloat(enbNode.lon), parseFloat(enbNode.lat)]);
                    
                    // 如果有展开的cluster，检查ENB是否在其中
                    if (this.expandedFeatures && this.expandedFeatures.length > 0) {
                        const expandedFeature = this.expandedFeatures.find(f => {
                            const expandedNodeData = f.get('nodeData');
                            // 比较节点的各种可能的标识符
                            return expandedNodeData && (
                                (expandedNodeData.code && expandedNodeData.code === enbNode.code) ||
                                (expandedNodeData.cellCode && expandedNodeData.cellCode === enbNode.cellCode) ||
                                (expandedNodeData.sn && expandedNodeData.sn === enbNode.sn)
                            );
                        });
                        
                        // 如果找到了展开的节点，使用展开后的坐标
                        if (expandedFeature) {
                            enbCoordinate = expandedFeature.getGeometry().getCoordinates();
                        }
                    }
                    
                    // 创建连线
                    const lineFeature = new ol.Feature({
                        geometry: new ol.geom.LineString([
                            ol.proj.fromLonLat([parseFloat(cpeNode.lon), parseFloat(cpeNode.lat)]),
                            enbCoordinate
                        ]),
                        cpeNode: cpeNode,
                        enbNode: enbNode
                    });

                    lineFeatures.push(lineFeature);
                    foundCount++;
                } else {
                    notFoundCount++;
                    if (notFoundCount <= 5) {
                        console.warn('Cannot create line for CPE:', cpeNode.code, 
                            'relaEnb:', cpeNode.relaEnb, 
                            'enbFound:', !!enbNode,
                            'cpeCoords valid:', !!(cpeNode.lat && cpeNode.lon),
                            'enbCoords valid:', enbNode ? !!(enbNode.lat && enbNode.lon) : false);
                    }
                }
            }
        });

        if (lineFeatures.length === 0) {
            console.warn('No lines created! Check if relaEnb values match ENB codes');
            return;
        }

        // 创建连线数据源和图层
        this.lineSource = new ol.source.Vector({
            features: lineFeatures
        });

        this.lineLayer = new ol.layer.Vector({
            source: this.lineSource,
            style: new ol.style.Style({
                stroke: new ol.style.Stroke({
                    color: 'rgba(64, 188, 51, 0.5)', // 绿色半透明连线
                    width: 2,
                    lineDash: [5, 5] // 虚线样式
                })
            }),
            // 设置较低的 zIndex，使连线显示在节点下方
            zIndex: 1
        });

        // 确保节点图层在连线上方
        if (this.vectorLayer) {
            this.vectorLayer.setZIndex(10);
        }

        if (this.map) {
            this.map.addLayer(this.lineLayer);
        }
    }

    /**
     * 根据节点数据和setting配置确定显示状态
     * @param {Object} node - 节点数据
     * @param {Object} statusForm - setting中的statusForm配置（包含deviceStatus、activeStatus、sasStatus）
     * @returns {String} 状态值：online/offline/inactive/registed/granted/authed
     */
    getNodeStatus(node, statusForm) {
        // 从全局window对象获取statusForm（如果没有传入）
        const effectiveStatusForm = statusForm || (typeof window !== 'undefined' ? window.topoStatusForm : null);
        
        // 如果有statusForm配置，根据配置的优先级判断
        if (effectiveStatusForm) {
            const configuredStatus = this._getStatusByConfig(node, effectiveStatusForm);
            if (configuredStatus) {
                return configuredStatus;
            }
        }
        
        // 如果没有statusForm配置，使用默认逻辑（向后兼容）
        return this._getDefaultStatus(node);
    }
    
    /**
     * 根据配置获取节点状态
     * @private
     * @param {Object} node - 节点数据
     * @param {Object} statusForm - statusForm配置
     * @returns {String|null} 状态值或null（如果无法确定）
     */
    _getStatusByConfig(node, statusForm) {
        // 优先使用 SAS 状态（如果sasStatus配置不为空）
        const sasStatus = this._getSasStatus(node, statusForm);
        if (sasStatus) {
            return sasStatus;
        }
        
        // 其次使用激活状态（如果activeStatus配置不为空）
        const activeStatus = this._getActiveStatus(node, statusForm);
        if (activeStatus) {
            return activeStatus;
        }
        
        // 最后使用设备在线状态（如果deviceStatus配置不为空）
        const deviceStatus = this._getDeviceStatus(node, statusForm);
        if (deviceStatus) {
            return deviceStatus;
        }
        
        return null;
    }
    
    /**
     * 获取SAS状态
     * @private
     */
    _getSasStatus(node, statusForm) {
        if (!this._isConfigEnabled(statusForm.sasStatus) || !node.sasState) {
            return null;
        }
        return this._mapSasState(node.sasState);
    }
    
    /**
     * 获取激活状态
     * @private
     */
    _getActiveStatus(node, statusForm) {
        if (!this._isConfigEnabled(statusForm.activeStatus)) {
            return null;
        }
        if (node.active === undefined || node.active === null) {
            return null;
        }
        return node.active == 'yes' ? 'online' : 'inactive';
    }
    
    /**
     * 获取设备在线状态
     * @private
     */
    _getDeviceStatus(node, statusForm) {
        if (!this._isConfigEnabled(statusForm.deviceStatus)) {
            return null;
        }
        if (node.online === undefined || node.online === null) {
            return null;
        }
        return node.online === 'on' ? 'online' : 'offline';
    }
    
    /**
     * 检查配置是否启用
     * @private
     */
    _isConfigEnabled(config) {
        return config && config.length > 0;
    }
    
    /**
     * 映射SAS状态
     * @private
     */
    _mapSasState(sasState) {
        const sasStateMap = {
            'Unregistered': 'inactive',
            'Registered': 'registed',
            'Granted': 'granted',
            'Authorized': 'authed'
        };
        return sasStateMap[sasState] || 'inactive';
    }
    
    /**
     * 获取默认状态（向后兼容）
     * @private
     */
    _getDefaultStatus(node) {
        // 优先使用 SAS 状态
        if (node.sasState) {
            return this._mapSasState(node.sasState);
        }
        
        // 其次使用在线状态
        if (node.online !== undefined && node.online !== null) {
            return node.online === 'on' ? 'online' : 'offline';
        }
        
        // 默认状态
        return 'online';
    }

    /**
     * 获取要素样式（带缓存优化）
     * @param {ol.Feature} feature - 要素对象
     */
    getFeatureStyle(feature) {
        // 检查是否是被隐藏的展开cluster的原始feature
        if (this.expandedCluster && this.expandedCluster.originalFeature === feature) {
            // 返回空样式数组，隐藏该feature
            return [];
        }
        
        const nodeType = feature.get('nodeType');
        const nodeStatus = feature.get('nodeStatus');
        const nodeData = feature.get('nodeData');
        const isCluster = feature.get('isCluster');
        const clusterSize = feature.get('clusterSize');
        
        // 生成样式缓存键，包含所有影响样式的因素
        const ueEnabled = window.topoUeStatusSettings && window.topoUeStatusSettings.includes('ue');
        const mdtEnabled = window.topoUeStatusSettings && window.topoUeStatusSettings.includes('mdt');
        const nameEnabled = window.topoNameStatusSettings && window.topoNameStatusSettings.includes('snName');
        const showUE = nodeData && nodeData.ueShow === true && nodeData.ueCount;
        const showMDT = nodeData && nodeData.hasMDT === true;
        
        // 获取KPI相关信息
        let kpiColor = null;
        let kpiValue = null;
        if (this.kpiMode && this.kpiStyles && nodeData) {
            const nodeCode = nodeData.code || nodeData.cellCode || nodeData.sn;
            const kpiStyle = this.kpiStyles[nodeCode];
            if (kpiStyle) {
                kpiColor = kpiStyle.color;
                kpiValue = kpiStyle.value;
            }
        }
        
        // 构建缓存键
        const cacheKey = [
            nodeType,
            nodeStatus,
            this.kpiMode ? 'kpi' : 'normal',
            kpiColor || 'none',
            kpiValue != null ? kpiValue : 'none',
            ueEnabled && showUE ? `ue:${nodeData.ueCount}` : 'noue',
            mdtEnabled && showMDT ? 'mdt' : 'nomdt',
            nameEnabled && nodeData ? `name:${nodeData.sn}` : 'noname',
            isCluster ? `cluster:${clusterSize}` : 'single'
        ].join('|');
        
        // 检查缓存
        const cachedStyle = feature.get('_styleCache');
        const cachedKey = feature.get('_styleCacheKey');
        
        if (cachedStyle && cachedKey === cacheKey) {
            // 返回缓存的样式
            return cachedStyle;
        }
        
        // 缓存未命中，重新计算样式
        const styles = this._computeFeatureStyle(feature, nodeType, nodeStatus, nodeData, isCluster, clusterSize);
        
        // 存储到缓存
        feature.set('_styleCache', styles, true); // true表示不触发change事件
        feature.set('_styleCacheKey', cacheKey, true);
        
        return styles;
    }

    /**
     * 计算要素样式（实际计算逻辑）
     * @private
     */
    _computeFeatureStyle(feature, nodeType, nodeStatus, nodeData, isCluster, clusterSize) {
        if (!nodeType || !nodeStatus) {
            console.warn('Missing nodeType or nodeStatus for feature:', {nodeType, nodeStatus, nodeData});
        }
        
        const styleConf = this._getStyleConfig(nodeType, nodeStatus, nodeData);
        const scale = this._getNodeScale(nodeType, styleConf.size);
        const styles = [];
        
        const geometry = feature.getGeometry();
        const coordinates = geometry.getCoordinates();
        const view = this.map.getView();
        const resolution = view.getResolution();
        const offsetY = 2 * resolution;
        
        // 添加基础节点样式（背景圆 + 图标）
        this._addBaseNodeStyles(styles, coordinates, offsetY, styleConf, nodeType, scale);
        
        // 添加 KPI 值标签
        this._addKpiValueLabel(styles, coordinates, offsetY, resolution, styleConf, nodeData);
        
        // 添加 UE/MDT 标签
        this._addUeMdtLabels(styles, coordinates, offsetY, resolution, styleConf, nodeData);
        
        // 添加 SN&Name 标签
        this._addSnNameLabel(styles, coordinates, offsetY, resolution, styleConf, nodeData);
        
        // 添加 cluster 标记（虚线框和徽章）
        this._addClusterMarker(styles, feature, coordinates, resolution, isCluster, clusterSize);
        
        return styles;
    }
    
    /**
     * 获取样式配置
     * @private
     */
    _getStyleConfig(nodeType, nodeStatus, nodeData) {
        if (this.kpiMode && this.kpiStyles && nodeData) {
            const nodeCode = nodeData.code || nodeData.cellCode || nodeData.sn;
            const kpiStyle = this.kpiStyles[nodeCode];
            
            if (kpiStyle) {
                return { 
                    color: kpiStyle.color, 
                    size: this.styleConfig[nodeType] ? this.styleConfig[nodeType][nodeStatus].size : 12
                };
            }
        }
        
        return this.styleConfig[nodeType] && this.styleConfig[nodeType][nodeStatus] 
            || { color: '#999', size: 10 };
    }
    
    /**
     * 获取节点缩放比例
     * @private
     */
    _getNodeScale(nodeType, size) {
        const scaleBase = (nodeType === 'gsm' || nodeType === 'gnb') ? 84 : 72;
        return size / scaleBase;
    }
    
    /**
     * 添加基础节点样式（背景圆 + 图标）
     * @private
     */
    _addBaseNodeStyles(styles, coordinates, offsetY, styleConf, nodeType, scale) {
        const backgroundPoint = new ol.geom.Point([
            coordinates[0],
            coordinates[1] + offsetY
        ]);
        
        const backgroundStyle = new ol.style.Style({
            geometry: backgroundPoint,
            image: new ol.style.Circle({
                radius: styleConf.size * 0.8,  // 稍微大一点，作为背景
                fill: new ol.style.Fill({
                    color: '#ffffff'
                }),
                stroke: new ol.style.Stroke({
                    color: '#e0e0e0',  // 浅灰色边框
                    width: 1
                })
            })
        });
        styles.push(backgroundStyle);
        
        // 添加节点图标
        const coloredSvgUrl = this.createColoredSvgUrl(nodeType, styleConf.color);
        const iconStyle = new ol.style.Style({
            image: new ol.style.Icon({
                src: coloredSvgUrl,
                scale: scale,
                anchor: [0.5, 0.5],
                anchorXUnits: 'fraction',
                anchorYUnits: 'fraction',
                opacity: 1
            })
        });
        styles.push(iconStyle);
    }
    
    /**
     * 添加 KPI 值标签
     * @private
     */
    _addKpiValueLabel(styles, coordinates, offsetY, resolution, styleConf, nodeData) {
        if (!this.kpiMode || !this.kpiShowValue || !this.kpiStyles || !nodeData) {
            return;
        }
        
        const nodeCode = nodeData.code || nodeData.cellCode || nodeData.sn;
        const kpiStyle = this.kpiStyles[nodeCode];
        
        if (!kpiStyle || kpiStyle.value == null) {
            return;
        }
        
        const kpiValue = kpiStyle.value.toString();
        const { bgDataUrl, width, height } = this._createRoundedRectCanvas(kpiValue, 'bold 11px Arial', 8, 20, 10);
        
        const kpiOffsetY = (styleConf.size * 0.8 + 15) * resolution;
        const kpiPoint = new ol.geom.Point([
            coordinates[0],
            coordinates[1] + offsetY + kpiOffsetY
        ]);
        
        styles.push(new ol.style.Style({
            geometry: kpiPoint,
            image: new ol.style.Icon({
                src: bgDataUrl,
                anchor: [0.5, 0.5],
                anchorXUnits: 'fraction',
                anchorYUnits: 'fraction'
            })
        }));
        
        styles.push(new ol.style.Style({
            geometry: kpiPoint,
            text: new ol.style.Text({
                text: kpiValue,
                font: 'bold 11px Arial',
                fill: new ol.style.Fill({ color: '#888' }),
                textAlign: 'center',
                textBaseline: 'middle',
                offsetX: 0,
                offsetY: 0
            })
        }));
    }
    
    /**
     * 创建圆角矩形背景 Canvas
     * @private
     */
    _createRoundedRectCanvas(text, font, padding, height, borderRadius) {
        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');
        
        ctx.font = font;
        const textWidth = ctx.measureText(text).width;
        const width = textWidth + padding * 2;
        
        canvas.width = width;
        canvas.height = height;
        
        ctx.fillStyle = 'rgba(255, 255, 255, 0.95)';
        ctx.strokeStyle = '#d0d0d0';
        ctx.lineWidth = 1;
        
        this._drawRoundedRect(ctx, 0, 0, width, height, borderRadius);
        ctx.fill();
        ctx.stroke();
        
        return { bgDataUrl: canvas.toDataURL(), width, height };
    }
    
    /**
     * 绘制圆角矩形路径
     * @private
     */
    _drawRoundedRect(ctx, x, y, width, height, radius) {
        ctx.beginPath();
        ctx.moveTo(x + radius, y);
        ctx.lineTo(x + width - radius, y);
        ctx.quadraticCurveTo(x + width, y, x + width, y + radius);
        ctx.lineTo(x + width, y + height - radius);
        ctx.quadraticCurveTo(x + width, y + height, x + width - radius, y + height);
        ctx.lineTo(x + radius, y + height);
        ctx.quadraticCurveTo(x, y + height, x, y + height - radius);
        ctx.lineTo(x, y + radius);
        ctx.quadraticCurveTo(x, y, x + radius, y);
        ctx.closePath();
    }
    
    /**
     * 添加 UE/MDT 标签
     * @private
     */
    _addUeMdtLabels(styles, coordinates, offsetY, resolution, styleConf, nodeData) {
        const showUE = nodeData && nodeData.ueShow === true && nodeData.ueCount;
        const showMDT = nodeData && nodeData.hasMDT === true;
        const ueStatusEnabled = window.topoUeStatusSettings && 
            (window.topoUeStatusSettings.includes('ue') || window.topoUeStatusSettings.includes('mdt'));
        const ueEnabled = window.topoUeStatusSettings && window.topoUeStatusSettings.includes('ue');
        const mdtEnabled = window.topoUeStatusSettings && window.topoUeStatusSettings.includes('mdt');
        
        if (!ueStatusEnabled || (!showUE && !showMDT)) {
            return;
        }
        
        const labelDistance = 8;
        const backgroundRadius = styleConf.size * 0.8;
        
        if (showUE && ueEnabled && (!showMDT || !mdtEnabled)) {
            this._addUeLabel(styles, coordinates, offsetY, resolution, backgroundRadius, labelDistance, nodeData.ueCount);
        } else if (showMDT && mdtEnabled && (!showUE || !ueEnabled)) {
            this._addMdtLabel(styles, coordinates, offsetY, resolution, backgroundRadius, labelDistance);
        } else if (showUE && ueEnabled && showMDT && mdtEnabled) {
            this._addUeMdtCombinedLabel(styles, coordinates, offsetY, resolution, backgroundRadius, labelDistance, nodeData.ueCount);
        }
    }
    
    /**
     * 添加 UE 数量标签
     * @private
     */
    _addUeLabel(styles, coordinates, offsetY, resolution, backgroundRadius, labelDistance, ueCount) {
        const labelText = ueCount.toString();
        const { bgDataUrl, width } = this._createRoundedRectCanvas(labelText, 'bold 10px Arial', 6, 18, 4);
        
        const labelOffsetX = (backgroundRadius + labelDistance + width / 2) * resolution;
        const labelPoint = new ol.geom.Point([
            coordinates[0] + labelOffsetX,
            coordinates[1] + offsetY
        ]);
        
        styles.push(new ol.style.Style({
            geometry: labelPoint,
            image: new ol.style.Icon({
                src: bgDataUrl,
                anchor: [0.5, 0.5],
                anchorXUnits: 'fraction',
                anchorYUnits: 'fraction'
            })
        }));
        
        styles.push(new ol.style.Style({
            geometry: labelPoint,
            text: new ol.style.Text({
                text: labelText,
                font: 'bold 10px Arial',
                fill: new ol.style.Fill({ color: '#333' }),
                textAlign: 'center',
                textBaseline: 'middle',
                offsetX: 0,
                offsetY: 0
            })
        }));
    }
    
    /**
     * 添加 MDT 图标标签
     * @private
     */
    _addMdtLabel(styles, coordinates, offsetY, resolution, backgroundRadius, labelDistance) {
        const mdtIconSize = 1;
        const padding = 6;
        const height = 18;
        const width = mdtIconSize + padding * 2;
        
        const canvas = document.createElement('canvas');
        canvas.width = width;
        canvas.height = height;
        
        const ctx = canvas.getContext('2d');
        ctx.fillStyle = 'rgba(255, 255, 255, 0.95)';
        ctx.strokeStyle = '#d0d0d0';
        ctx.lineWidth = 1;
        
        this._drawRoundedRect(ctx, 0, 0, width, height, 4);
        ctx.fill();
        ctx.stroke();
        
        const labelOffsetX = (backgroundRadius + labelDistance + width / 2) * resolution;
        const labelPoint = new ol.geom.Point([
            coordinates[0] + labelOffsetX,
            coordinates[1] + offsetY
        ]);
        
        styles.push(new ol.style.Style({
            geometry: labelPoint,
            image: new ol.style.Icon({
                src: canvas.toDataURL(),
                anchor: [0.5, 0.5],
                anchorXUnits: 'fraction',
                anchorYUnits: 'fraction'
            })
        }));
        
        const mdtSvgUrl = this.mdtSvgCache || '/js/ol/images/icon-MDT.svg';
        styles.push(new ol.style.Style({
            geometry: labelPoint,
            image: new ol.style.Icon({
                src: mdtSvgUrl,
                scale: mdtIconSize / 24,
                anchor: [0.5, 0.5],
                anchorXUnits: 'fraction',
                anchorYUnits: 'fraction'
            })
        }));
        
        if (!this.mdtSvgCache) {
            this._loadMdtSvgAsync();
        }
    }
    
    /**
     * 添加 UE + MDT 组合标签
     * @private
     */
    _addUeMdtCombinedLabel(styles, coordinates, offsetY, resolution, backgroundRadius, labelDistance, ueCount) {
        const ueText = ueCount.toString();
        const mdtIconSize = 1;
        const separator = ' | ';
        const mdtIconMargin = 2;
        
        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');
        ctx.font = 'bold 10px Arial';
        
        const ueTextWidth = ctx.measureText(ueText).width;
        const separatorWidth = ctx.measureText(separator).width;
        const totalContentWidth = ueTextWidth + separatorWidth + mdtIconMargin + mdtIconSize;
        
        const padding = 8;
        const height = 18;
        const width = totalContentWidth + padding * 2;
        
        canvas.width = width;
        canvas.height = height;
        
        ctx.fillStyle = 'rgba(255, 255, 255, 0.95)';
        ctx.strokeStyle = '#d0d0d0';
        ctx.lineWidth = 1;
        
        this._drawRoundedRect(ctx, 0, 0, width, height, 4);
        ctx.fill();
        ctx.stroke();
        
        const labelOffsetX = (backgroundRadius + labelDistance + width / 2) * resolution;
        const labelPoint = new ol.geom.Point([
            coordinates[0] + labelOffsetX,
            coordinates[1] + offsetY
        ]);
        
        styles.push(new ol.style.Style({
            geometry: labelPoint,
            image: new ol.style.Icon({
                src: canvas.toDataURL(),
                anchor: [0.5, 0.5],
                anchorXUnits: 'fraction',
                anchorYUnits: 'fraction'
            })
        }));
        
        // UE 文本
        const ueTextOffsetX = -(totalContentWidth / 2) + (ueTextWidth / 2);
        styles.push(new ol.style.Style({
            geometry: labelPoint,
            text: new ol.style.Text({
                text: ueText,
                font: 'bold 10px Arial',
                fill: new ol.style.Fill({ color: '#333' }),
                textAlign: 'center',
                textBaseline: 'middle',
                offsetX: ueTextOffsetX,
                offsetY: 0
            })
        }));
        
        // 分隔符
        const separatorOffsetX = -(totalContentWidth / 2) + ueTextWidth + (separatorWidth / 2);
        styles.push(new ol.style.Style({
            geometry: labelPoint,
            text: new ol.style.Text({
                text: separator,
                font: 'bold 10px Arial',
                fill: new ol.style.Fill({ color: '#333' }),
                textAlign: 'center',
                textBaseline: 'middle',
                offsetX: separatorOffsetX,
                offsetY: 0
            })
        }));
        
        // MDT 图标
        const mdtIconOffsetX = (totalContentWidth / 2) - (mdtIconSize / 2);
        const mdtIconPoint = new ol.geom.Point([
            labelPoint.getCoordinates()[0] + mdtIconOffsetX * resolution,
            labelPoint.getCoordinates()[1]
        ]);
        
        const mdtSvgUrl = this.mdtSvgCache || '/js/ol/images/icon-MDT.svg';
        styles.push(new ol.style.Style({
            geometry: mdtIconPoint,
            image: new ol.style.Icon({
                src: mdtSvgUrl,
                scale: mdtIconSize / 24,
                anchor: [0.5, 0.5],
                anchorXUnits: 'fraction',
                anchorYUnits: 'fraction'
            })
        }));
        
        if (!this.mdtSvgCache) {
            this._loadMdtSvgAsync();
        }
    }
    
    /**
     * 异步加载 MDT SVG
     * @private
     */
    _loadMdtSvgAsync() {
        this.pendingSvgLoads++;
        this.loadMdtSvg().then(() => {
            this.pendingSvgLoads--;
            if (this.pendingSvgLoads === 0) {
                this.debouncedRefresh(100);
            }
        });
    }
    
    /**
     * 添加 SN&Name 标签
     * @private
     */
    _addSnNameLabel(styles, coordinates, offsetY, resolution, styleConf, nodeData) {
        const showSnName = nodeData && window.topoNameStatusSettings && 
            window.topoNameStatusSettings.includes('snName');
        
        if (!showSnName) {
            return;
        }
        
        const sn = nodeData.sn || nodeData.serial_number || nodeData.code || '';
        const name = nodeData.name || nodeData.cellName || '';
        
        const textLines = [];
        if (sn) textLines.push('SN: ' + sn);
        if (name) textLines.push('Cell Name: ' + name);
        
        if (textLines.length === 0) {
            return;
        }
        
        const combinedDataUrl = this._createSnNameCanvas(textLines);
        const { bgHeight } = this._calculateSnNameDimensions(textLines);
        
        const labelDistance = 8;
        const backgroundRadius = styleConf.size * 0.8;
        const labelOffsetY = (backgroundRadius + labelDistance + bgHeight / 2) * resolution;
        const labelPoint = new ol.geom.Point([
            coordinates[0],
            coordinates[1] + offsetY + labelOffsetY
        ]);
        
        styles.push(new ol.style.Style({
            geometry: labelPoint,
            image: new ol.style.Icon({
                src: combinedDataUrl,
                anchor: [0.5, 0.5],
                anchorXUnits: 'fraction',
                anchorYUnits: 'fraction'
            }),
            zIndex: 20
        }));
    }
    
    /**
     * 创建 SN&Name Canvas
     * @private
     */
    _createSnNameCanvas(textLines) {
        const canvas = document.createElement('canvas');
        const ctx = canvas.getContext('2d');
        
        const fontSize = 11;
        const fontFamily = 'Arial';
        ctx.font = `${fontSize}px ${fontFamily}`;
        
        const lineHeight = 14;
        const padding = 6;
        const borderRadius = 10;
        
        let maxTextWidth = 0;
        textLines.forEach(line => {
            const width = ctx.measureText(line).width;
            if (width > maxTextWidth) maxTextWidth = width;
        });
        
        const bgWidth = maxTextWidth + padding * 2;
        const bgHeight = (lineHeight * textLines.length) + padding * 2;
        
        canvas.width = bgWidth;
        canvas.height = bgHeight;
        
        ctx.font = `${fontSize}px ${fontFamily}`;
        ctx.textAlign = 'left';
        ctx.textBaseline = 'middle';
        
        ctx.fillStyle = 'rgba(255, 255, 255, 0.95)';
        ctx.strokeStyle = '#d0d0d0';
        ctx.lineWidth = 1;
        
        this._drawRoundedRect(ctx, 0, 0, bgWidth, bgHeight, borderRadius);
        ctx.fill();
        ctx.stroke();
        
        ctx.fillStyle = '#333';
        const startY = padding + lineHeight / 2;
        const textX = padding;
        textLines.forEach((line, index) => {
            const y = startY + (index * lineHeight);
            ctx.fillText(line, textX, y);
        });
        
        return canvas.toDataURL();
    }
    
    /**
     * 计算 SN&Name 标签尺寸
     * @private
     */
    _calculateSnNameDimensions(textLines) {
        const lineHeight = 14;
        const padding = 6;
        const bgHeight = (lineHeight * textLines.length) + padding * 2;
        return { bgHeight };
    }
    
    /**
     * 添加 Cluster 标记（虚线框和徽章）
     * @private
     */
    _addClusterMarker(styles, feature, coordinates, resolution, isCluster, clusterSize) {
        if (!isCluster || clusterSize <= 1) {
            return;
        }
        
        const squareSize = 40;
        const halfSize = squareSize / 2;
        const squareSizeInCoords = halfSize * resolution;
        
        // 创建虚线正方形
        const squareCoords = [
            [coordinates[0] - squareSizeInCoords, coordinates[1] + squareSizeInCoords],
            [coordinates[0] + squareSizeInCoords, coordinates[1] + squareSizeInCoords],
            [coordinates[0] + squareSizeInCoords, coordinates[1] - squareSizeInCoords],
            [coordinates[0] - squareSizeInCoords, coordinates[1] - squareSizeInCoords],
            [coordinates[0] - squareSizeInCoords, coordinates[1] + squareSizeInCoords]
        ];
        
        const squareGeometry = new ol.geom.LineString(squareCoords);
        styles.push(new ol.style.Style({
            geometry: squareGeometry,
            stroke: new ol.style.Stroke({
                color: '#1890ff',
                width: 1,
                lineDash: [4, 4]
            })
        }));
        
        // 添加徽章
        const badgeText = '+' + clusterSize.toString();
        const badgeBgDataUrl = this._createBadgeCanvas(badgeText);
        
        const badgeOffsetX = squareSizeInCoords;
        const badgeOffsetY = squareSizeInCoords;
        const badgePoint = new ol.geom.Point([
            coordinates[0] + badgeOffsetX,
            coordinates[1] - badgeOffsetY
        ]);
        
        styles.push(new ol.style.Style({
            geometry: badgePoint,
            image: new ol.style.Icon({
                src: badgeBgDataUrl,
                anchor: [0.5, 0.5],
                anchorXUnits: 'fraction',
                anchorYUnits: 'fraction'
            })
        }));
        
        styles.push(new ol.style.Style({
            geometry: badgePoint,
            text: new ol.style.Text({
                text: badgeText,
                font: 'bold 12px Arial',
                fill: new ol.style.Fill({ color: '#1890ff' }),
                textAlign: 'center',
                textBaseline: 'middle',
                offsetX: 0,
                offsetY: 0
            })
        }));
    }
    
    /**
     * 创建徽章 Canvas
     * @private
     */
    _createBadgeCanvas(badgeText) {
        const badgeCanvas = document.createElement('canvas');
        const badgeCtx = badgeCanvas.getContext('2d');
        
        badgeCtx.font = 'bold 12px Arial';
        const badgeTextWidth = badgeCtx.measureText(badgeText).width;
        
        const badgePadding = 4;
        const badgeHeight = 16;
        const badgeWidth = badgeTextWidth + badgePadding * 2;
        
        badgeCanvas.width = badgeWidth;
        badgeCanvas.height = badgeHeight;
        
        badgeCtx.fillStyle = 'rgba(255, 255, 255, 0.95)';
        badgeCtx.strokeStyle = '#d0d0d0';
        badgeCtx.lineWidth = 1;
        
        this._drawRoundedRect(badgeCtx, 0, 0, badgeWidth, badgeHeight, 3);
        badgeCtx.fill();
        badgeCtx.stroke();
        
        return badgeCanvas.toDataURL();
    }

    /**
     * 清除所有节点的样式缓存
     * 当显示设置（如SN&Name、UE、MDT等）改变时，需要清除缓存以强制重新渲染
     */
    clearStyleCache() {
        if (this.vectorSource) {
            const features = this.vectorSource.getFeatures();
            features.forEach(feature => {
                feature.unset('_styleCache', true);
                feature.unset('_styleCacheKey', true);
            });
        }
    }

    /**
     * 添加鼠标悬浮事件（支持鼠标移动到悬浮框上）
     * @param {Function} callback - 悬浮回调函数
     */
    addHoverInteraction(callback) {
        let currentFeature = null;
        let hideTimeout = null;
        let isOverPopup = false;

        this.map.on('pointermove', (evt) => {
            if (evt.dragging) {
                return;
            }

            // 如果鼠标在浮层上，不处理地图上的pointermove事件
            if (isOverPopup) {
                return;
            }

            const pixel = this.map.getEventPixel(evt.originalEvent);
            const feature = this.map.forEachFeatureAtPixel(pixel, (feature) => feature);

            if (feature !== currentFeature) {
                // 清除之前的延迟隐藏
                if (hideTimeout) {
                    clearTimeout(hideTimeout);
                    hideTimeout = null;
                }
                
                if (currentFeature && !feature) {
                    // 鼠标离开节点，延迟隐藏以便鼠标可以移到悬浮框上
                    hideTimeout = setTimeout(() => {
                        if (!isOverPopup) {
                            callback(null, evt);
                            currentFeature = null;
                        }
                    }, 200); // 200ms 延迟
                } else if (feature) {
                    // 鼠标进入新节点
                    const nodeData = feature.get('nodeData');
                    callback(nodeData, evt, feature);
                    currentFeature = feature;
                }
            }

            // 更改鼠标样式
            this.map.getTargetElement().style.cursor = feature ? 'pointer' : '';
        });

        // 为悬浮框添加鼠标事件监听
        const setupPopupListeners = () => {
            if (this.overlay && this.overlay.getElement()) {
                const popupElement = this.overlay.getElement();
                
                popupElement.addEventListener('mouseenter', () => {
                    isOverPopup = true;
                    if (hideTimeout) {
                        clearTimeout(hideTimeout);
                        hideTimeout = null;
                    }
                });
                
                popupElement.addEventListener('mouseleave', () => {
                    isOverPopup = false;
                    // 鼠标离开悬浮框，隐藏它
                    callback(null, null);
                    currentFeature = null;
                });
            }
        };

        // 延迟设置监听器，等待 overlay 创建
        setTimeout(setupPopupListeners, 100);
        
        // 保存设置函数供外部调用
        this.setupPopupListeners = setupPopupListeners;
    }

    /**
     * 添加点击事件
     * @param {Function} callback - 点击回调函数
     */
    addClickInteraction(callback) {
        this.map.on('click', (evt) => {
            // 如果刚完成拖拽，不触发点击事件
            if (this.justDragged) {
                return;
            }
            
            const feature = this.map.forEachFeatureAtPixel(evt.pixel, (feature) => feature);
            
            if (feature) {
                const featureType = feature.get('featureType');
                
                // 如果点击的是小区扇面（白色固定大小），展开信号覆盖扇面
                if (featureType === 'cellSector') {
                    const nodeData = feature.get('nodeData');
                    if (nodeData) {
                        this.drawSignalCoverageSector(nodeData);
                    }
                    return; // 不调用回调，避免重复显示详情面板
                }
                
                // 如果点击的是信号覆盖扇面（蓝色渐变），收起它
                if (featureType === 'signalCoverage') {
                    this.clearSignalCoverageSector();
                    return;
                }
                
                // 否则是点击节点，先清除所有扇面，然后调用回调
                this.clearAllSectors();
                const nodeData = feature.get('nodeData');
                this.selectedFeature = feature;
                callback(nodeData, evt, feature);
            } else {
                // 点击空白区域，清除所有扇面和高亮
                this.clearAllSectors();
                this.clearHighlight();
                callback(null, evt);
            }
        });
    }

    /**
     * 检查节点是否有绘制扇面所需的数据
     */
    hasRequiredSectorData(nodeData) {
        return nodeData 
            && nodeData.mechanical_downtilt != null
            && nodeData.electronic_downtilt != null
            && nodeData.vertical_3dB_beam_width != null
            && nodeData.direct != null
            && nodeData.height != null;
    }

    /**
     * 创建悬浮提示层
     * @param {HTMLElement} element - 提示框元素
     */
    createOverlay(element) {
        this.overlay = new ol.Overlay({
            element: element,
            autoPan: false, // 禁用自动平移 - 悬浮提示不应该移动地图
            positioning: 'bottom-center',
            stopEvent: true, // 阻止事件传播，防止点击浮层时穿透到下面的节点
            offset: [0, -10]
        });
        
        this.map.addOverlay(this.overlay);
        
        // 创建后立即设置悬浮框的事件监听器
        if (this.setupPopupListeners) {
            setTimeout(() => this.setupPopupListeners(), 50);
        }
        
        return this.overlay;
    }

    /**
     * 显示悬浮提示
     * @param {Array} coordinate - 坐标 [lon, lat]
     * @param {Object} content - 内容数据
     */
    showOverlay(coordinate, content) {
        if (this.overlay) {
            const position = ol.proj.fromLonLat(coordinate);
            this.overlay.setPosition(position);
            
            // 更新内容由外部处理
            return this.overlay.getElement();
        }
    }

    /**
     * 隐藏悬浮提示
     */
    hideOverlay() {
        if (this.overlay) {
            this.overlay.setPosition(undefined);
        }
    }

    /**
     * 更新节点数据 (高性能更新)
     * @param {Array} nodes - 新的节点数据
     */
    updateNodes(nodes) {
        if (!this.vectorSource) return;

        // 缓存所有节点数据，用于更新连线
        this.allNodesCache = nodes;

        // 使用增量更新策略
        this.vectorSource.clear();
        
        // 清空并重建重叠节点映射
        this.clusterMap.clear();
        
        // 第一步：按经纬度分组节点
        const locationMap = new Map();
        nodes.forEach(node => {
            if (!node.lat || !node.lon || isNaN(node.lat) || isNaN(node.lon)) {
                return;
            }
            
            const locKey = `${node.lat},${node.lon}`;
            if (!locationMap.has(locKey)) {
                locationMap.set(locKey, []);
            }
            locationMap.get(locKey).push(node);
        });
        
        // 第二步：为每个位置创建 Feature，并记录重叠信息
        const features = [];
        locationMap.forEach((nodesAtLocation, locKey) => {
            // 使用第一个节点作为代表节点
            const representativeNode = nodesAtLocation[0];
            
            // 确定节点类型
            let nodeType = representativeNode.type || 'enb';
            if (representativeNode.type === 'enb' && representativeNode.isGSM) {
                nodeType = 'gsm';
            }

            // 确定节点状态（从全局获取statusForm配置）
            const statusForm = (typeof window !== 'undefined' && window.topoStatusForm) ? window.topoStatusForm : null;
            let nodeStatus = this.getNodeStatus(representativeNode, statusForm);

            const feature = new ol.Feature({
                geometry: new ol.geom.Point(
                    ol.proj.fromLonLat([parseFloat(representativeNode.lon), parseFloat(representativeNode.lat)])
                ),
                nodeData: representativeNode, // 代表节点数据
                allNodes: nodesAtLocation, // 该位置的所有节点
                nodeType: nodeType,
                nodeStatus: nodeStatus,
                isCluster: nodesAtLocation.length > 1, // 是否为重叠节点
                clusterSize: nodesAtLocation.length // 重叠数量
            });

            features.push(feature);
            
            // 记录重叠节点分组
            if (nodesAtLocation.length > 1) {
                this.clusterMap.set(locKey, nodesAtLocation);
            }
        });

        this.vectorSource.addFeatures(features);
        
        // 如果有展开的cluster，需要更新 originalFeature 的引用（因为 features 已经重新创建）
        if (this.expandedCluster && this.expandedCluster.locKey) {
            // 在新的 features 中找到对应的 cluster feature
            const newClusterFeature = features.find(f => {
                const nodeData = f.get('nodeData');
                if (!nodeData) return false;
                const locKey = nodeData.lat + ',' + nodeData.lon;
                return locKey === this.expandedCluster.locKey;
            });
            
            if (newClusterFeature) {
                // 更新引用
                this.expandedCluster.originalFeature = newClusterFeature;
            } else {
                // 如果找不到对应的 cluster，说明该 cluster 已经不存在了，收起展开
                this.collapseCluster(true);
            }
        }
        
        // 同时更新连线图层
        this.updateLines(nodes);
    }

    /**
     * 更新 CPE 到 ENB 的连线（在节点更新时调用）
     * @param {Array} allNodes - 所有节点数据
     */
    updateLines(allNodes) {
        // 过滤出有 relaEnb 的 CPE 节点
        const cpeNodesWithRelaEnb = allNodes.filter(node => 
            node.type === 'cpe' && node.relaEnb
        );
        
        if (cpeNodesWithRelaEnb.length > 0) {
            this.createLinesLayer(cpeNodesWithRelaEnb, allNodes);
        } else if (this.lineLayer && this.map) {
            // 如果没有连线数据，移除连线图层
            this.map.removeLayer(this.lineLayer);
            this.lineLayer = null;
            this.lineSource = null;
        }
    }

    /**
     * 过滤节点（只渲染可视区域内的节点 - 性能优化）
     * @param {Array} allNodes - 所有节点
     * @param {Number} threshold - 阈值，超过此数量才启用过滤
     */
    filterVisibleNodes(allNodes, threshold = 5000) {
        if (allNodes.length <= threshold) {
            this.updateNodes(allNodes);
            return;
        }

        const extent = this.map.getView().calculateExtent();
        const visibleNodes = allNodes.filter(node => {
            const coord = ol.proj.fromLonLat([parseFloat(node.lon), parseFloat(node.lat)]);
            return ol.extent.containsCoordinate(extent, coord);
        });

        this.updateNodes(visibleNodes);
    }

    /**
     * 添加缩放和移动结束事件监听
     * @param {Function} callback - 回调函数
     */
    addViewChangeListeners(callback) {
        this.map.getView().on('change:resolution', () => {
            if (callback) callback('zoom');
        });

        this.map.on('moveend', () => {
            if (callback) callback('move');
        });
    }

    /**
     * 销毁地图
     */
    destroy() {
        if (this.map) {
            // 清理所有图层引用
            if (this.vectorLayer) {
                this.map.removeLayer(this.vectorLayer);
                this.vectorLayer = null;
                this.vectorSource = null;
            }
            
            if (this.lineLayer) {
                this.map.removeLayer(this.lineLayer);
                this.lineLayer = null;
                this.lineSource = null;
            }
            
            if (this.highlightLayer) {
                this.map.removeLayer(this.highlightLayer);
                this.highlightLayer = null;
            }
            
            if (this.expandedCluster && this.expandedCluster.expandLayer) {
                this.map.removeLayer(this.expandedCluster.expandLayer);
                this.expandedCluster = null;
            }
            
            // 销毁地图
            this.map.setTarget(null);
            this.map = null;
        }
    }

    /**
     * 适应边界
     * @param {Array} bounds - [minLat, minLon, maxLat, maxLon]
     */
    fitBounds(bounds) {
        const extent = ol.proj.transformExtent(
            [bounds[1], bounds[0], bounds[3], bounds[2]], // [minLon, minLat, maxLon, maxLat]
            'EPSG:4326',
            'EPSG:3857'
        );
        
        this.map.getView().fit(extent, {
            padding: [50, 50, 50, 50],
            duration: 500
        });
    }

    /**
     * 定位到指定节点并高亮显示
     * @param {Object} node - 节点数据（包含 lat, lon）
     * @param {Number} zoom - 缩放级别，默认18
     * @param {Boolean} keepExpanded - 是否保持展开状态（不收起cluster）
     */
    locateAndHighlightNode(node, zoom = 18, keepExpanded = false) {
        if (!node || !node.lat || !node.lon) {
            console.warn('Invalid node data for location');
            return;
        }

        // 如果正在执行定位操作，先清除之前的操作
        if (this.isLocatingAndHighlighting) {
            console.log('Previous location operation in progress, clearing...');
            this.isLocatingAndHighlighting = false;
            // 清除可能存在的旧高亮
            this.clearHighlight();
        }

        // 设置标志，表示正在执行定位操作
        this.isLocatingAndHighlighting = true;

        // 检查节点是否在展开的cluster中
        let targetFeature = null;
        if (this.expandedFeatures && this.expandedFeatures.length > 0) {
            targetFeature = this.expandedFeatures.find(f => {
                const nodeData = f.get('nodeData');
                return nodeData && nodeData.code === node.code;
            });
        }

        // 如果节点不在已展开的cluster中，检查是否在其他cluster中
        if (!targetFeature) {
            const locKey = node.lat + ',' + node.lon;
            const clusterNodes = this.clusterMap.get(locKey);
            
            // 如果该位置有多个节点（是cluster），需要先缩放再展开
            if (clusterNodes && clusterNodes.length > 1) {
                console.log('Node is in a cluster, will expand after zoom. Cluster size:', clusterNodes.length);
                
                // 找到对应的cluster feature
                const clusterFeature = this.vectorSource.getFeatures().find(f => {
                    const nodeData = f.get('nodeData');
                    const isMatch = nodeData && 
                        Math.abs(nodeData.lat - node.lat) < 0.0001 && 
                        Math.abs(nodeData.lon - node.lon) < 0.0001 && 
                        f.get('isCluster');
                    return isMatch;
                });
                
                if (clusterFeature) {
                    this._zoomAndExpandCluster(node, zoom, clusterFeature);
                    return; // 提前返回，因为后续操作在回调中完成
                } else {
                    this._zoomAndSearchCluster(node, zoom);
                    return; // 提前返回
                }
            }
        }

        // 设置保持展开状态的标志
        if (keepExpanded && this.expandedCluster) {
            this.keepExpandedOnViewChange = true;
        } else {
            this.keepExpandedOnViewChange = false;
        }

        let coordinate;
        if (targetFeature) {
            // 如果节点已展开，使用展开后的位置
            coordinate = targetFeature.getGeometry().getCoordinates();
        } else {
            // 使用原始位置
            coordinate = ol.proj.fromLonLat([parseFloat(node.lon), parseFloat(node.lat)]);
        }
        
        // 不缩放，只移动到中心（如果保持展开状态）
        if (keepExpanded && this.expandedCluster) {
            // 平滑移动到节点位置，但不改变缩放级别
            this.map.getView().animate({
                center: coordinate,
                duration: 500
            }, () => {
                // 动画完成后清除标志
                this.isLocatingAndHighlighting = false;
            });
        } else {
            // 平滑缩放到节点位置
            this.map.getView().animate({
                center: coordinate,
                zoom: zoom,
                duration: 800
            }, () => {
                // 动画完成后清除标志
                this.isLocatingAndHighlighting = false;
            });
        }

        // 高亮显示节点（延迟一点，确保动画开始后再高亮）
        setTimeout(() => {
            this.highlightNode(node, coordinate);
        }, 50);
    }

    /**
     * 缩放并展开cluster（找到cluster feature的情况）
     * @private
     */
    _zoomAndExpandCluster(node, zoom, clusterFeature) {
        console.log('Cluster feature found, starting zoom and expand sequence');
        const centerCoord = ol.proj.fromLonLat([parseFloat(node.lon), parseFloat(node.lat)]);
        
        // 设置自动定位标志，防止缩放过程中收起cluster
        this.isAutoLocating = true;
        
        // 先缩放到目标位置，这样展开时的resolution才正确
        this.map.getView().animate({
            center: centerCoord,
            zoom: zoom,
            duration: 600
        }, (complete) => {
            if (!complete) {
                // 缩放被中断，清除标志
                this.isAutoLocating = false;
                this.isLocatingAndHighlighting = false;
                return;
            }
            
            console.log('Zoom completed, now expanding cluster');
            // 缩放完成后再展开cluster
            this.expandCluster(clusterFeature);
            
            // 等待展开动画和渲染完成，然后定位到具体节点
            setTimeout(() => {
                const targetFeature = this.expandedFeatures.find(f => {
                    const nodeData = f.get('nodeData');
                    return nodeData && nodeData.code === node.code;
                });
                
                if (targetFeature) {
                    const coordinate = targetFeature.getGeometry().getCoordinates();
                    
                    // 平滑移动到展开后的节点位置（不再缩放，只移动）
                    this.map.getView().animate({
                        center: coordinate,
                        duration: 400
                    });
                    
                    // 延迟高亮，确保节点已经渲染完成
                    setTimeout(() => {
                        this.highlightNode(node, coordinate);
                    }, 50);
                } else {
                    console.warn('Target feature not found after cluster expansion');
                }
                
                // 清除自动定位标志
                this.isAutoLocating = false;
                this.isLocatingAndHighlighting = false;
            }, 200);
        });
    }

    /**
     * 缩放并搜索cluster（未找到cluster feature的情况）
     * @private
     */
    _zoomAndSearchCluster(node, zoom) {
        // 找不到cluster feature，可能节点不在可视区域
        // 先移动到目标位置，等地图刷新后再尝试展开
        console.log('Cluster feature not found initially, moving to position first');
        const centerCoord = ol.proj.fromLonLat([parseFloat(node.lon), parseFloat(node.lat)]);
        
        this.isAutoLocating = true;
        
        this.map.getView().animate({
            center: centerCoord,
            zoom: zoom,
            duration: 600
        }, (complete) => {
            this._handleZoomCompleteForSearch(complete, node, centerCoord);
        });
    }

    /**
     * 处理缩放完成后的搜索逻辑
     * @private
     */
    _handleZoomCompleteForSearch(complete, node, centerCoord) {
        if (!complete) {
            this.isAutoLocating = false;
            this.isLocatingAndHighlighting = false;
            return;
        }
        
        console.log('Zoom completed, now searching for cluster feature');
        // 地图刷新后，重新查找cluster feature
        setTimeout(() => {
            this._searchAndExpandCluster(node, centerCoord);
        }, 100);
    }

    /**
     * 搜索并展开cluster
     * @private
     */
    _searchAndExpandCluster(node, centerCoord) {
        const clusterFeatureAfterZoom = this.vectorSource.getFeatures().find(f => {
            const nodeData = f.get('nodeData');
            const isMatch = nodeData && 
                Math.abs(nodeData.lat - node.lat) < 0.0001 && 
                Math.abs(nodeData.lon - node.lon) < 0.0001 && 
                f.get('isCluster');
            return isMatch;
        });
        
        if (clusterFeatureAfterZoom) {
            this._expandAndHighlightNode(clusterFeatureAfterZoom, node);
        } else {
            this._highlightNodeAtCenter(node, centerCoord);
        }
    }

    /**
     * 展开cluster并高亮目标节点
     * @private
     */
    _expandAndHighlightNode(clusterFeature, node) {
        console.log('Cluster feature found after zoom, expanding');
        this.expandCluster(clusterFeature);
        
        // 等待展开完成后高亮节点
        setTimeout(() => {
            const targetFeature = this.expandedFeatures.find(f => {
                const nodeData = f.get('nodeData');
                return nodeData && nodeData.code === node.code;
            });
            
            if (targetFeature) {
                const coordinate = targetFeature.getGeometry().getCoordinates();
                this.map.getView().animate({
                    center: coordinate,
                    duration: 400
                });
                
                setTimeout(() => {
                    this.highlightNode(node, coordinate);
                }, 50);
            }
            
            this.isAutoLocating = false;
            this.isLocatingAndHighlighting = false;
        }, 200);
    }

    /**
     * 在中心位置高亮节点（cluster未找到的情况）
     * @private
     */
    _highlightNodeAtCenter(node, centerCoord) {
        console.warn('Cluster feature still not found after zoom');
        // 即使找不到，也高亮原始位置
        setTimeout(() => {
            this.highlightNode(node, centerCoord);
        }, 50);
        this.isAutoLocating = false;
        this.isLocatingAndHighlighting = false;
    }

    /**
     * 扇形展开重叠节点
     * @param {ol.Feature} clusterFeature - cluster feature
     */
    expandCluster(clusterFeature) {
        if (!this.map) return;
        
        const allNodes = clusterFeature.get('allNodes');
        const locKey = clusterFeature.get('nodeData').lat + ',' + clusterFeature.get('nodeData').lon;
        
        if (!allNodes || allNodes.length <= 1) {
            return;
        }

        // 如果已经有展开的cluster，先收起
        if (this.expandedCluster && this.expandedCluster.locKey === locKey) {
            this.collapseCluster(true);
            return;
        } else if (this.expandedCluster) {
            this.collapseCluster(true);
        }

        // 获取中心坐标
        const centerCoord = clusterFeature.getGeometry().getCoordinates();
        
        // 计算扇形展开的位置
        const radius = 50; // 展开半径（像素）
        const resolution = this.map.getView().getResolution();
        const radiusInMapUnits = radius * resolution;
        
        // 计算每个节点的角度
        const angleStep = (2 * Math.PI) / allNodes.length;
        const startAngle = -Math.PI / 2; // 从正上方开始
        
        // 创建展开后的features
        const expandedFeatures = [];
        
        allNodes.forEach((node, index) => {
            const angle = startAngle + angleStep * index;
            const offsetX = radiusInMapUnits * Math.cos(angle);
            const offsetY = radiusInMapUnits * Math.sin(angle);
            
            const expandedCoord = [
                centerCoord[0] + offsetX,
                centerCoord[1] + offsetY
            ];
            
            // 确定节点类型
            let nodeType = node.type || 'enb';
            if (node.type === 'enb' && node.isGSM) {
                nodeType = 'gsm';
            }
            
            // 确定节点状态（从全局获取statusForm配置）
            const statusForm = (typeof window !== 'undefined' && window.topoStatusForm) ? window.topoStatusForm : null;
            let nodeStatus = this.getNodeStatus(node, statusForm);
            
            const feature = new ol.Feature({
                geometry: new ol.geom.Point(expandedCoord),
                nodeData: node,
                nodeType: nodeType,
                nodeStatus: nodeStatus,
                isExpanded: true, // 标记为展开状态
                originalCluster: locKey // 记录原始cluster
            });
            
            expandedFeatures.push(feature);
        });
        
        // 创建展开图层
        const expandLayer = new ol.layer.Vector({
            source: new ol.source.Vector({
                features: expandedFeatures
            }),
            style: (feature) => this.getFeatureStyle(feature),
            zIndex: 50 // 在普通节点之上，但在高亮层之下
        });
        
        if (this.map) {
            this.map.addLayer(expandLayer);
        }
        
        // 保存展开状态（必须在隐藏原始feature之前设置，因为getFeatureStyle会检查这个值）
        this.expandedCluster = {
            locKey: locKey,
            expandLayer: expandLayer,
            originalFeature: clusterFeature
        };
        
        this.expandedFeatures = expandedFeatures;
        
        // 设置保持展开状态标志，防止视图变化时自动收起
        this.keepExpandedOnViewChange = true;
        
        // 触发原始图层重新渲染，使getFeatureStyle生效，隐藏原始cluster
        this.vectorSource.changed();
        
        // 更新连线，使其连接到展开后的节点位置
        if (this.allNodesCache && this.allNodesCache.length > 0) {
            this.updateLines(this.allNodesCache);
        }
    }

    /**
     * 收起展开的节点
     * @param {Boolean} force - 是否强制收起（忽略 keepExpandedOnViewChange 和 isAutoLocating 标志）
     */
    collapseCluster(force = false) {
        if (!this.expandedCluster || !this.map) {
            return;
        }
        
        // 如果正在自动定位中且不是强制收起，则不执行收起操作
        if (this.isAutoLocating && !force) {
            return;
        }
        
        // 如果设置了保持展开状态且不是强制收起，则不执行收起操作
        if (this.keepExpandedOnViewChange && !force) {
            return;
        }
        
        // 移除展开图层
        if (this.expandedCluster.expandLayer && this.map) {
            this.map.removeLayer(this.expandedCluster.expandLayer);
        }
        
        // 清除高亮效果（收起cluster时，高亮的节点可能是展开的节点，需要清除）
        this.clearHighlight();
        
        // 清空展开状态（getFeatureStyle会自动恢复显示原始cluster）
        this.expandedCluster = null;
        this.expandedFeatures = [];
        this.keepExpandedOnViewChange = false; // 重置标志
        
        // 触发原始图层重新渲染，使原始cluster重新显示
        this.vectorSource.changed();
        
        // 更新连线，恢复到原始cluster位置
        if (this.allNodesCache && this.allNodesCache.length > 0) {
            this.updateLines(this.allNodesCache);
        }
    }

    /**
     * 高亮显示指定节点（渐变白色圆形背景）
     * @param {Object} node - 节点数据
     * @param {Array} coordinate - 可选的坐标（用于展开后的节点）
     */
    highlightNode(node, coordinate) {
        if (!node || !this.map) return;

        // 移除旧的高亮图层（确保只有一个节点高亮）
        if (this.highlightLayer && this.map) {
            this.map.removeLayer(this.highlightLayer);
            this.highlightLayer = null;
        }

        // 如果没有传入坐标，使用节点的原始坐标
        if (!coordinate) {
            if (!node.lat || !node.lon) return;
            coordinate = ol.proj.fromLonLat([parseFloat(node.lon), parseFloat(node.lat)]);
        }
        
        // 创建高亮 Feature
        const highlightFeature = new ol.Feature({
            geometry: new ol.geom.Point(coordinate)
        });

        // 创建多个同心圆，实现向外淡化的渐变效果
        const circles = [];
        const maxRadius = 20; // 最大半径
        const minRadius = 8; // 最小半径
        const steps = 2; // 渐变层数
        
        for (let i = 0; i < steps; i++) {
            const ratio = (steps - i) / steps;
            const radius = minRadius + (maxRadius - minRadius) * ratio;
            const opacity = 0.6 * (1 - i / steps); // 从0.6逐渐淡化到0
            
            circles.push(new ol.style.Style({
                image: new ol.style.Circle({
                    radius: radius,
                    fill: new ol.style.Fill({
                        color: `rgba(255, 255, 255, ${opacity})`
                    }),
                    stroke: i === steps - 1 ? new ol.style.Stroke({
                        color: 'rgba(255, 255, 255, 0.3)',
                        width: 1
                    }) : null
                })
            }));
        }

        // 创建高亮图层，zIndex设置为5，位于节点图层(zIndex: 10)下方
        this.highlightLayer = new ol.layer.Vector({
            source: new ol.source.Vector({
                features: [highlightFeature]
            }),
            style: circles,
            zIndex: 5 // 在节点图层(10)下方，在连线图层(1)上方
        });

        if (this.map) {
            this.map.addLayer(this.highlightLayer);
        }
    }

    /**
     * 清除节点高亮
     */
    clearHighlight() {
        if (this.highlightLayer && this.map) {
            this.map.removeLayer(this.highlightLayer);
            this.highlightLayer = null;
        }
    }

    /**
     * 获取重叠节点信息
     * @param {String} locKey - 位置键 "lat,lon"
     * @returns {Array} 该位置的所有节点
     */
    getClusterNodes(locKey) {
        return this.clusterMap.get(locKey) || [];
    }

    /**
     * 根据节点 code 查找节点
     * @param {String} code - 节点代码
     * @param {Array} allNodes - 所有节点数组
     * @returns {Object|null} 找到的节点
     */
    findNodeByCode(code, allNodes) {
        return allNodes.find(node => 
            node.code === code || 
            node.cellCode === code || 
            node.sn === code
        ) || null;
    }

    /**
     * 搜索节点（模糊匹配）
     * @param {String} keyword - 搜索关键词
     * @param {Array} allNodes - 所有节点数组
     * @returns {Array} 匹配的节点数组
     * 
     * 支持的搜索字段：
     * - code: 节点代码（实际为序列号SN）
     * - name: 节点名称
     * - cellCode: 小区代码
     * - ip: IP地址
     * - sn: 序列号简写
     * - serial_number: 序列号（下划线格式）
     * - serialNumber: 序列号（驼峰格式）
     * - SerialNumber: 序列号（首字母大写）
     */
    searchNodes(keyword, allNodes) {
        if (!keyword || keyword.trim() === '') {
            return [];
        }

        const lowerKeyword = keyword.toLowerCase().trim();
        
        const results = allNodes.filter(node => {
            // 将所有字段转换为字符串并转小写进行比较
            const code = node.code ? String(node.code).toLowerCase() : '';
            const name = node.name ? String(node.name).toLowerCase() : '';
            //const cellCode = node.cellCode ? String(node.cellCode).toLowerCase() : '';
            const ip = node.ip ? String(node.ip) : '';
            const sn = node.sn ? String(node.sn).toLowerCase() : '';
            const serialNumber1 = node.serial_number ? String(node.serial_number).toLowerCase() : '';
            const serialNumber2 = node.serialNumber ? String(node.serialNumber).toLowerCase() : '';
            
            return code.includes(lowerKeyword) || 
                   name.includes(lowerKeyword) || 
                   //cellCode.includes(lowerKeyword) || 
                   ip.includes(keyword.trim()) || 
                   sn.includes(lowerKeyword) || 
                   serialNumber1.includes(lowerKeyword) || 
                   serialNumber2.includes(lowerKeyword);
        });
        
        return results;
    }

    /**
     * 开始测距功能
     */
    startMeasure() {
        if (!this.map) {
            console.error('Map not initialized');
            return;
        }

        // 如果已经在测距，先停止
        if (this.measureActive) {
            this.stopMeasure();
        }

        // 强制清理旧的事件监听器（防止页面重新加载后残留）
        if (this.measureClickHandler) {
            this.map.un('click', this.measureClickHandler);
            this.measureClickHandler = null;
        }
        if (this.measureMoveHandler) {
            this.map.un('pointermove', this.measureMoveHandler);
            this.measureMoveHandler = null;
        }
        if (this.measureDblClickHandler) {
            this.map.un('dblclick', this.measureDblClickHandler);
            this.measureDblClickHandler = null;
        }

        // 清理旧的图层和标签
        if (this.measureSource) {
            this.measureSource.clear();
        }
        if (this.measureTooltips) {
            this.measureTooltips.forEach(tooltip => {
                if (this.map) {
                    this.map.removeOverlay(tooltip);
                }
            });
        }
        if (this.measureTempTooltip) {
            this.map.removeOverlay(this.measureTempTooltip);
            this.measureTempTooltip = null;
        }
        
        // 移除旧图层并重新创建（确保干净状态）
        if (this.measureLayer && this.map) {
            this.map.removeLayer(this.measureLayer);
            this.measureLayer = null;
            this.measureSource = null;
        }

        this.measureActive = true;
        this.measureCoordinates = [];
        
        // 创建测距图层（总是重新创建以确保干净状态）
        this.measureSource = new ol.source.Vector();
        this.measureLayer = new ol.layer.Vector({
            source: this.measureSource,
            style: new ol.style.Style({
                fill: new ol.style.Fill({
                    color: 'rgba(255, 255, 255, 0.2)'
                }),
                stroke: new ol.style.Stroke({
                    color: '#3399ff',
                    width: 3,
                    lineDash: [10, 5]
                }),
                image: new ol.style.Circle({
                    radius: 5,
                    fill: new ol.style.Fill({
                        color: '#3399ff'
                    }),
                    stroke: new ol.style.Stroke({
                        color: '#fff',
                        width: 2
                    })
                })
            }),
            zIndex: 200
        });
        
        if (this.map) {
            this.map.addLayer(this.measureLayer);
        }

        // 创建临时线（跟随鼠标）
        this.measureTempFeature = null;
        
        // 创建测量标签的Overlay
        this.measureTooltips = [];

        // 绑定点击事件
        this.measureClickHandler = this.handleMeasureClick.bind(this);
        this.measureMoveHandler = this.handleMeasureMove.bind(this);
        this.measureDblClickHandler = this.handleMeasureDblClick.bind(this);

        this.map.on('click', this.measureClickHandler);
        this.map.on('pointermove', this.measureMoveHandler);
        this.map.on('dblclick', this.measureDblClickHandler);

        // 禁用双击缩放
        this.map.getInteractions().forEach(interaction => {
            if (interaction instanceof ol.interaction.DoubleClickZoom) {
                interaction.setActive(false);
            }
        });

        // 改变鼠标样式
        this.map.getViewport().style.cursor = 'crosshair';
    }

    /**
     * 处理测距点击事件
     */
    handleMeasureClick(evt) {
        if (!this.measureActive) return;

        const coordinate = evt.coordinate;
        this.measureCoordinates.push(coordinate);

        // 添加点标记
        const pointFeature = new ol.Feature({
            geometry: new ol.geom.Point(coordinate)
        });
        this.measureSource.addFeature(pointFeature);

        // 如果有多个点，绘制线段并显示距离
        if (this.measureCoordinates.length > 1) {
            const line = new ol.geom.LineString(this.measureCoordinates);
            
            // 移除旧的线要素（保留点）
            const features = this.measureSource.getFeatures();
            features.forEach(feature => {
                if (feature.getGeometry().getType() === 'LineString') {
                    this.measureSource.removeFeature(feature);
                }
            });

            // 添加新的线要素
            const lineFeature = new ol.Feature({
                geometry: line
            });
            this.measureSource.addFeature(lineFeature);

            // 计算总距离
            const length = this.calculateDistance(line);
            
            // 创建距离标签
            this.createMeasureTooltip(coordinate, length, true);
        }
    }

    /**
     * 处理鼠标移动事件（显示临时线）
     */
    handleMeasureMove(evt) {
        if (!this.measureActive || this.measureCoordinates.length === 0) return;

        const coordinate = evt.coordinate;
        
        // 移除临时线和临时标签
        if (this.measureTempFeature) {
            this.measureSource.removeFeature(this.measureTempFeature);
        }
        if (this.measureTempTooltip) {
            this.map.removeOverlay(this.measureTempTooltip);
        }

        // 创建临时线
        const tempCoords = [...this.measureCoordinates, coordinate];
        const tempLine = new ol.geom.LineString(tempCoords);
        this.measureTempFeature = new ol.Feature({
            geometry: tempLine
        });
        this.measureSource.addFeature(this.measureTempFeature);

        // 显示临时距离
        if (this.measureCoordinates.length > 0) {
            const length = this.calculateDistance(tempLine);
            this.createMeasureTooltip(coordinate, length, false);
        }
    }

    /**
     * 处理双击事件（结束测距）
     */
    handleMeasureDblClick(evt) {
        if (!this.measureActive) return;
        
        evt.preventDefault();
        evt.stopPropagation();
        
        // 移除临时线和临时标签
        if (this.measureTempFeature) {
            this.measureSource.removeFeature(this.measureTempFeature);
        }
        if (this.measureTempTooltip) {
            this.map.removeOverlay(this.measureTempTooltip);
        }

        // 解绑事件但保持测量结果
        this.map.un('click', this.measureClickHandler);
        this.map.un('pointermove', this.measureMoveHandler);
        this.map.un('dblclick', this.measureDblClickHandler);

        // 恢复双击缩放
        this.map.getInteractions().forEach(interaction => {
            if (interaction instanceof ol.interaction.DoubleClickZoom) {
                interaction.setActive(true);
            }
        });

        // 恢复鼠标样式
        this.map.getViewport().style.cursor = '';
        
        this.measureActive = false;
    }

    /**
     * 停止测距并清除所有测量结果
     */
    stopMeasure() {
        if (!this.map) return;

        // 解绑事件
        if (this.measureClickHandler) {
            this.map.un('click', this.measureClickHandler);
        }
        if (this.measureMoveHandler) {
            this.map.un('pointermove', this.measureMoveHandler);
        }
        if (this.measureDblClickHandler) {
            this.map.un('dblclick', this.measureDblClickHandler);
        }

        // 清除测量图层
        if (this.measureSource) {
            this.measureSource.clear();
        }

        // 清除所有标签
        if (this.measureTooltips) {
            this.measureTooltips.forEach(tooltip => {
                this.map.removeOverlay(tooltip);
            });
            this.measureTooltips = [];
        }

        // 清除临时标签
        if (this.measureTempTooltip) {
            this.map.removeOverlay(this.measureTempTooltip);
            this.measureTempTooltip = null;
        }

        // 移除测量图层
        if (this.measureLayer && this.map) {
            this.map.removeLayer(this.measureLayer);
            this.measureLayer = null;
            this.measureSource = null;
        }

        // 恢复双击缩放
        this.map.getInteractions().forEach(interaction => {
            if (interaction instanceof ol.interaction.DoubleClickZoom) {
                interaction.setActive(true);
            }
        });

        // 恢复鼠标样式
        this.map.getViewport().style.cursor = '';

        this.measureActive = false;
        this.measureCoordinates = [];
        this.measureTempFeature = null;
    }

    /**
     * 创建测量标签
     */
    createMeasureTooltip(coordinate, distance, isPermanent) {
        const tooltipElement = document.createElement('div');
        tooltipElement.className = 'ol-measure-tooltip';
        
        const textSpan = document.createElement('span');
        textSpan.className = 'measure-text';
        textSpan.textContent = this.formatDistance(distance);
        
        if (isPermanent) {
            const closeBtn = document.createElement('span');
            closeBtn.className = 'measure-close el-icon-close';
            closeBtn.onclick = (e) => {
                e.stopPropagation();
                this.stopMeasure();
            };
            tooltipElement.appendChild(textSpan);
            tooltipElement.appendChild(closeBtn);
            tooltipElement.classList.add('measure-tooltip-permanent');
        } else {
            tooltipElement.appendChild(textSpan);
            tooltipElement.classList.add('measure-tooltip-temp');
        }

        const tooltip = new ol.Overlay({
            element: tooltipElement,
            offset: [0, -15],
            positioning: 'bottom-center',
            stopEvent: false,
            insertFirst: false
        });

        tooltip.setPosition(coordinate);
        this.map.addOverlay(tooltip);

        if (isPermanent) {
            this.measureTooltips.push(tooltip);
        } else {
            this.measureTempTooltip = tooltip;
        }
    }

    /**
     * 计算距离（米）
     */
    calculateDistance(line) {
        const coordinates = line.getCoordinates();
        let length = 0;
        
        for (let i = 0; i < coordinates.length - 1; i++) {
            const c1 = ol.proj.toLonLat(coordinates[i]);
            const c2 = ol.proj.toLonLat(coordinates[i + 1]);
            length += this.getDistanceFromLatLon(c1[1], c1[0], c2[1], c2[0]);
        }
        
        return length * 1000; // 转换为米
    }

    /**
     * 计算两点间距离（Haversine公式）
     */
    getDistanceFromLatLon(lat1, lon1, lat2, lon2) {
        const R = 6371; // 地球半径（公里）
        const dLat = this.deg2rad(lat2 - lat1);
        const dLon = this.deg2rad(lon2 - lon1);
        const a = 
            Math.sin(dLat / 2) * Math.sin(dLat / 2) +
            Math.cos(this.deg2rad(lat1)) * Math.cos(this.deg2rad(lat2)) *
            Math.sin(dLon / 2) * Math.sin(dLon / 2);
        const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
        const d = R * c;
        return d;
    }

    deg2rad(deg) {
        return deg * (Math.PI / 180);
    }

    /**
     * 格式化距离显示
     */
    formatDistance(meters) {
        if (meters < 1000) {
            return meters.toFixed(2) + ' m';
        } else {
            return (meters / 1000).toFixed(2) + ' km';
        }
    }

    /**
     * 开始 Site Map 测距功能
     */
    startMeasureSite() {
        if (!window.globSiteMap) {
            console.error('Site Map not initialized');
            return;
        }

        // 如果已经在测距，先停止
        if (this.measureSiteActive) {
            this.stopMeasureSite();
        }

        // 强制清理旧的事件监听器（防止页面重新加载后残留）
        if (this.measureSiteClickHandler) {
            window.globSiteMap.un('click', this.measureSiteClickHandler);
            this.measureSiteClickHandler = null;
        }
        if (this.measureSiteMoveHandler) {
            window.globSiteMap.un('pointermove', this.measureSiteMoveHandler);
            this.measureSiteMoveHandler = null;
        }
        if (this.measureSiteDblClickHandler) {
            window.globSiteMap.un('dblclick', this.measureSiteDblClickHandler);
            this.measureSiteDblClickHandler = null;
        }

        // 清理旧的图层和标签
        if (this.measureSiteSource) {
            this.measureSiteSource.clear();
        }
        if (this.measureSiteTooltips) {
            this.measureSiteTooltips.forEach(tooltip => {
                if (window.globSiteMap) {
                    window.globSiteMap.removeOverlay(tooltip);
                }
            });
        }
        if (this.measureSiteTempTooltip) {
            window.globSiteMap.removeOverlay(this.measureSiteTempTooltip);
            this.measureSiteTempTooltip = null;
        }
        
        // 移除旧图层并重新创建（确保干净状态）
        if (this.measureSiteLayer && window.globSiteMap) {
            window.globSiteMap.removeLayer(this.measureSiteLayer);
            this.measureSiteLayer = null;
            this.measureSiteSource = null;
        }

        this.measureSiteActive = true;
        this.measureSiteCoordinates = [];
        
        // 创建测距图层（总是重新创建以确保干净状态）
        this.measureSiteSource = new ol.source.Vector();
        this.measureSiteLayer = new ol.layer.Vector({
            source: this.measureSiteSource,
            style: new ol.style.Style({
                fill: new ol.style.Fill({
                    color: 'rgba(255, 255, 255, 0.2)'
                }),
                stroke: new ol.style.Stroke({
                    color: '#3399ff',
                    width: 3,
                    lineDash: [10, 5]
                }),
                image: new ol.style.Circle({
                    radius: 5,
                    fill: new ol.style.Fill({
                        color: '#3399ff'
                    }),
                    stroke: new ol.style.Stroke({
                        color: '#fff',
                        width: 2
                    })
                })
            }),
            zIndex: 200
        });
        
        window.globSiteMap.addLayer(this.measureSiteLayer);

        // 创建临时线（跟随鼠标）
        this.measureSiteTempFeature = null;
        
        // 创建测量标签的Overlay
        this.measureSiteTooltips = [];

        // 绑定点击事件
        this.measureSiteClickHandler = this.handleMeasureSiteClick.bind(this);
        this.measureSiteMoveHandler = this.handleMeasureSiteMove.bind(this);
        this.measureSiteDblClickHandler = this.handleMeasureSiteDblClick.bind(this);

        window.globSiteMap.on('click', this.measureSiteClickHandler);
        window.globSiteMap.on('pointermove', this.measureSiteMoveHandler);
        window.globSiteMap.on('dblclick', this.measureSiteDblClickHandler);

        // 禁用双击缩放
        window.globSiteMap.getInteractions().forEach(interaction => {
            if (interaction instanceof ol.interaction.DoubleClickZoom) {
                interaction.setActive(false);
            }
        });

        // 改变鼠标样式
        window.globSiteMap.getViewport().style.cursor = 'crosshair';
    }

    /**
     * 处理 Site Map 测距点击事件
     */
    handleMeasureSiteClick(evt) {
        if (!this.measureSiteActive) return;

        const coordinate = evt.coordinate;
        this.measureSiteCoordinates.push(coordinate);

        // 添加点标记
        const pointFeature = new ol.Feature({
            geometry: new ol.geom.Point(coordinate)
        });
        this.measureSiteSource.addFeature(pointFeature);

        // 如果有多个点，绘制线段并显示距离
        if (this.measureSiteCoordinates.length > 1) {
            const line = new ol.geom.LineString(this.measureSiteCoordinates);
            
            // 移除旧的线要素（保留点）
            const features = this.measureSiteSource.getFeatures();
            features.forEach(feature => {
                if (feature.getGeometry().getType() === 'LineString') {
                    this.measureSiteSource.removeFeature(feature);
                }
            });

            // 添加新的线要素
            const lineFeature = new ol.Feature({
                geometry: line
            });
            this.measureSiteSource.addFeature(lineFeature);

            // 计算总距离
            const length = this.calculateDistance(line);
            
            // 创建距离标签
            this.createMeasureSiteTooltip(coordinate, length, true);
        }
    }

    /**
     * 处理 Site Map 鼠标移动事件（显示临时线）
     */
    handleMeasureSiteMove(evt) {
        if (!this.measureSiteActive || this.measureSiteCoordinates.length === 0) return;

        const coordinate = evt.coordinate;
        
        // 移除临时线和临时标签
        if (this.measureSiteTempFeature) {
            this.measureSiteSource.removeFeature(this.measureSiteTempFeature);
        }
        if (this.measureSiteTempTooltip) {
            window.globSiteMap.removeOverlay(this.measureSiteTempTooltip);
        }

        // 创建临时线
        const tempCoords = [...this.measureSiteCoordinates, coordinate];
        const tempLine = new ol.geom.LineString(tempCoords);
        this.measureSiteTempFeature = new ol.Feature({
            geometry: tempLine
        });
        this.measureSiteSource.addFeature(this.measureSiteTempFeature);

        // 显示临时距离
        if (this.measureSiteCoordinates.length > 0) {
            const length = this.calculateDistance(tempLine);
            this.createMeasureSiteTooltip(coordinate, length, false);
        }
    }

    /**
     * 处理 Site Map 双击事件（结束测距）
     */
    handleMeasureSiteDblClick(evt) {
        if (!this.measureSiteActive) return;
        
        evt.preventDefault();
        evt.stopPropagation();
        
        // 移除临时线和临时标签
        if (this.measureSiteTempFeature) {
            this.measureSiteSource.removeFeature(this.measureSiteTempFeature);
        }
        if (this.measureSiteTempTooltip) {
            window.globSiteMap.removeOverlay(this.measureSiteTempTooltip);
        }

        // 解绑事件但保持测量结果
        window.globSiteMap.un('click', this.measureSiteClickHandler);
        window.globSiteMap.un('pointermove', this.measureSiteMoveHandler);
        window.globSiteMap.un('dblclick', this.measureSiteDblClickHandler);

        // 恢复双击缩放
        window.globSiteMap.getInteractions().forEach(interaction => {
            if (interaction instanceof ol.interaction.DoubleClickZoom) {
                interaction.setActive(true);
            }
        });

        // 恢复鼠标样式
        window.globSiteMap.getViewport().style.cursor = '';
        
        this.measureSiteActive = false;
    }

    /**
     * 停止 Site Map 测距并清除所有测量结果
     */
    stopMeasureSite() {
        if (!window.globSiteMap) return;

        // 解绑事件
        if (this.measureSiteClickHandler) {
            window.globSiteMap.un('click', this.measureSiteClickHandler);
        }
        if (this.measureSiteMoveHandler) {
            window.globSiteMap.un('pointermove', this.measureSiteMoveHandler);
        }
        if (this.measureSiteDblClickHandler) {
            window.globSiteMap.un('dblclick', this.measureSiteDblClickHandler);
        }

        // 清除测量图层
        if (this.measureSiteSource) {
            this.measureSiteSource.clear();
        }

        // 清除所有标签
        if (this.measureSiteTooltips) {
            this.measureSiteTooltips.forEach(tooltip => {
                window.globSiteMap.removeOverlay(tooltip);
            });
            this.measureSiteTooltips = [];
        }

        // 清除临时标签
        if (this.measureSiteTempTooltip) {
            window.globSiteMap.removeOverlay(this.measureSiteTempTooltip);
            this.measureSiteTempTooltip = null;
        }

        // 移除测量图层
        if (this.measureSiteLayer && window.globSiteMap) {
            window.globSiteMap.removeLayer(this.measureSiteLayer);
            this.measureSiteLayer = null;
            this.measureSiteSource = null;
        }

        // 恢复双击缩放
        window.globSiteMap.getInteractions().forEach(interaction => {
            if (interaction instanceof ol.interaction.DoubleClickZoom) {
                interaction.setActive(true);
            }
        });

        // 恢复鼠标样式
        window.globSiteMap.getViewport().style.cursor = '';
        
        this.measureSiteActive = false;
        this.measureSiteCoordinates = [];
        this.measureSiteTempFeature = null;
    }

    /**
     * 清理所有测距资源（在地图销毁前调用）
     */
    cleanupMeasureResources() {
        // 清理 Topo Map 测距
        if (this.measureActive) {
            this.stopMeasure();
        }
        
        // 清理 Site Map 测距
        if (this.measureSiteActive) {
            this.stopMeasureSite();
        }
        
        // 重置所有测距相关变量
        this.measureActive = false;
        this.measureCoordinates = [];
        this.measureSource = null;
        this.measureLayer = null;
        this.measureTempFeature = null;
        this.measureTooltips = [];
        this.measureTempTooltip = null;
        this.measureClickHandler = null;
        this.measureMoveHandler = null;
        this.measureDblClickHandler = null;
        
        this.measureSiteActive = false;
        this.measureSiteCoordinates = [];
        this.measureSiteSource = null;
        this.measureSiteLayer = null;
        this.measureSiteTempFeature = null;
        this.measureSiteTooltips = [];
        this.measureSiteTempTooltip = null;
        this.measureSiteClickHandler = null;
        this.measureSiteMoveHandler = null;
        this.measureSiteDblClickHandler = null;
    }

    /**
     * 启用节点拖拽功能
     * @param {boolean} enable - 是否启用拖拽（通常基于sasEnable的反值）
     * @param {function} dragEndCallback - 拖拽结束回调函数 (feature, originalCoords, newCoords)
     */
    enableNodeDrag(enable, dragEndCallback) {
        if (!this.map || !this.vectorLayer) {
            console.warn('Map or vector layer not initialized');
            return;
        }
        
        // 移除旧的拖拽交互
        if (this.translateInteraction) {
            this.map.removeInteraction(this.translateInteraction);
            this.translateInteraction = null;
        }
        
        this.isDragEnabled = enable;
        this.dragEndCallback = dragEndCallback;
        
        if (!enable) {
            return; // 不启用拖拽
        }
        
        // 创建拖拽交互
        this.translateInteraction = new ol.interaction.Translate({
            layers: [this.vectorLayer], // 只对节点图层生效
            hitTolerance: 5
        });
        
        // 记录拖拽前的原始坐标
        let originalCoords = null;
        let draggedFeature = null;
        let hasMoved = false; // 标记是否真正移动过
        
        this.translateInteraction.on('translatestart', (evt) => {
            // 如果正在测距，不允许拖拽
            if (this.measureActive) {
                evt.preventDefault();
                return;
            }
            
            this.isDragging = true;
            hasMoved = false;
            draggedFeature = evt.features.getArray()[0];
            if (draggedFeature) {
                const geometry = draggedFeature.getGeometry();
                originalCoords = geometry.getCoordinates();
            }
            
            // 隐藏悬浮提示，避免提示停留在原位置
            this.hideOverlay();
        });
        
        this.translateInteraction.on('translating', (evt) => {
            // 标记已经移动
            hasMoved = true;
            
            // 强制重新渲染被拖拽的 feature，使其样式（包括独立 geometry）跟随移动
            const features = evt.features.getArray();
            features.forEach(feature => {
                // 关键：清除样式缓存，强制 getFeatureStyle 重新计算样式
                feature.unset('_styleCache', true);
                feature.unset('_styleCacheKey', true);
                
                // 清除 feature 级别的样式，使用 layer 的样式函数
                feature.setStyle(null);
                
                // 标记 feature 的几何对象已改变（触发样式重新计算）
                feature.getGeometry().changed();
                
                // 标记 feature 已改变
                feature.changed();
            });
            
            // 强制图层和数据源重新渲染
            this.vectorSource.changed();
            this.vectorLayer.changed();
            
            // 强制地图立即渲染（同步渲染，不等待下一帧）
            this.map.renderSync();
        });
        
        this.translateInteraction.on('translateend', (evt) => {
            this.isDragging = false;
            
            if (!draggedFeature || !originalCoords) {
                return;
            }
            
            const geometry = draggedFeature.getGeometry();
            const newCoords = geometry.getCoordinates();
            
            // 检查是否真正移动过
            if (!hasMoved) {
                // 没有移动，不触发回调
                originalCoords = null;
                draggedFeature = null;
                return;
            }
            
            // 设置标志，防止后续的click事件触发
            this.justDragged = true;
            setTimeout(() => {
                this.justDragged = false;
            }, 300); // 300ms后清除标志
            
            // 转换为经纬度
            const originalLonLat = ol.proj.toLonLat(originalCoords);
            const newLonLat = ol.proj.toLonLat(newCoords);
            
            // 调用回调函数
            if (this.dragEndCallback) {
                this.dragEndCallback(draggedFeature, originalLonLat, newLonLat, originalCoords);
            }
            
            // 重置状态
            originalCoords = null;
            draggedFeature = null;
            hasMoved = false;
        });
        
        this.map.addInteraction(this.translateInteraction);
    }

    /**
     * 禁用节点拖拽功能
     */
    disableNodeDrag() {
        this.enableNodeDrag(false, null);
    }

    /**
     * 恢复Feature到原始位置
     * @param {ol.Feature} feature - 要恢复的feature
     * @param {Array} originalCoords - 原始坐标 [x, y] (已投影)
     */
    restoreFeaturePosition(feature, originalCoords) {
        if (feature && originalCoords) {
            const geometry = feature.getGeometry();
            geometry.setCoordinates(originalCoords);
            
            // 清除样式缓存，强制重新计算样式（确保背景和标签也回到原位置）
            feature.unset('_styleCache', true);
            feature.unset('_styleCacheKey', true);
            
            // 清除 feature 级别的样式
            feature.setStyle(null);
            
            // 标记几何对象和 feature 已改变
            geometry.changed();
            feature.changed();
            
            // 强制图层和地图重新渲染
            if (this.vectorSource) {
                this.vectorSource.changed();
            }
            if (this.vectorLayer) {
                this.vectorLayer.changed();
            }
            if (this.map) {
                this.map.renderSync();
            }
        }
    }

    /**
```     * 创建 Site Map 测量标签
     */
    createMeasureSiteTooltip(coordinate, distance, isPermanent) {
        const tooltipElement = document.createElement('div');
        tooltipElement.className = 'ol-measure-tooltip';
        
        const textSpan = document.createElement('span');
        textSpan.className = 'measure-text';
        textSpan.textContent = this.formatDistance(distance);
        
        if (isPermanent) {
            const closeBtn = document.createElement('span');
            closeBtn.className = 'measure-close el-icon-close';
            closeBtn.onclick = (e) => {
                e.stopPropagation();
                this.stopMeasureSite();
            };
            tooltipElement.appendChild(textSpan);
            tooltipElement.appendChild(closeBtn);
            tooltipElement.classList.add('measure-tooltip-permanent');
        } else {
            tooltipElement.appendChild(textSpan);
            tooltipElement.classList.add('measure-tooltip-temp');
        }

        const tooltip = new ol.Overlay({
            element: tooltipElement,
            offset: [0, -15],
            positioning: 'bottom-center',
            stopEvent: false,
            insertFirst: false
        });

        tooltip.setPosition(coordinate);
        window.globSiteMap.addOverlay(tooltip);

        if (isPermanent) {
            this.measureSiteTooltips.push(tooltip);
        } else {
            this.measureSiteTempTooltip = tooltip;
        }
    }

    /**
     * 检查节点是否具有绘制扇面所需的数据
     * @param {Object} nodeData - 节点数据
     * @returns {Boolean} 是否具有必需数据
     */
    hasRequiredSectorData(nodeData) {
        if (!nodeData) {
            return false;
        }
        
        const required = ['lat', 'lon', 'height', 'mechanical_downtilt', 'electronic_downtilt', 
                         'vertical_3dB_beam_width', 'direct'];
        
        const missing = [];
        for (let field of required) {
            if (nodeData[field] === undefined || nodeData[field] === null || nodeData[field] === '') {
                missing.push(field);
            }
        }
        
        if (missing.length > 0) {
            return false;
        }
        
        // 检查radius和minRadius
        if (!nodeData.radius || nodeData.radius <= 0) {
            // 标记需要重新计算
        }
        
        if (!nodeData.minRadius || nodeData.minRadius <= 0) {
            // 标记需要重新计算
        }
        
        return true;
    }

    /**
     * 计算扇面覆盖半径
     * @param {Object} nodeData - 节点数据
     * @returns {Object} {radius, minRadius} 计算后的半径
     */
    calculateSectorRadius(nodeData) {
        const {
            height,
            mechanical_downtilt,
            electronic_downtilt,
            vertical_3dB_beam_width
        } = nodeData;

        let radius = 500;  // 默认500米
        let minRadius = 100; // 默认100米

        if (height && mechanical_downtilt && electronic_downtilt && vertical_3dB_beam_width) {
            const totalDowntilt = mechanical_downtilt * 1 + electronic_downtilt * 1;
            const halfBeamWidth = vertical_3dB_beam_width / 2;
            
            // 外半径：下倾角 - 半功率波束宽度的一半
            const outerAngle = totalDowntilt - halfBeamWidth;
            // 内半径：下倾角 + 半功率波束宽度的一半
            const innerAngle = totalDowntilt + halfBeamWidth;
            
            // 确保角度在合理范围内（避免tan值异常）
            if (outerAngle > 0 && outerAngle < 90) {
                const outerRad = (outerAngle * Math.PI) / 180;
                radius = Math.abs(height / Math.tan(outerRad));
            } else {
                radius = 500;
            }
            
            if (innerAngle > 0 && innerAngle < 90) {
                const innerRad = (innerAngle * Math.PI) / 180;
                minRadius = Math.abs(height / Math.tan(innerRad));
            } else {
                minRadius = 100;
            }
            
            // 限制半径在合理范围内
            if (radius > 10000) radius = 10000; // 最大10km
            if (radius < 50) radius = 50;       // 最小50m
            if (minRadius > radius) minRadius = radius * 0.2; // 内半径不应大于外半径
            if (minRadius < 10) minRadius = 10;   // 最小10m
            
            // 四舍五入保留2位小数
            radius = parseFloat(radius.toFixed(2));
            minRadius = parseFloat(minRadius.toFixed(2));
        }

        return { radius, minRadius };
    }

    /**
     * 绘制节点选中效果（实心蓝色圆圈）
     * @param {Object} nodeData - 节点数据
     */
    drawNodeSelection(nodeData) {
        const { lat, lon } = nodeData;
        
        if (!lat || !lon) return;
        
        // 移除旧的选中标记
        if (this.nodeSelectionLayer) {
            this.map.removeLayer(this.nodeSelectionLayer);
        }
        
        const center = ol.proj.fromLonLat([parseFloat(lon), parseFloat(lat)]);
        
        // 创建圆形几何（实心蓝色圆圈）
        const circleFeature = new ol.Feature({
            geometry: new ol.geom.Point(center),
            featureType: 'nodeSelection'
        });
        
        // 设置样式：实心蓝色圆圈，稍微向上偏移
        circleFeature.setStyle(new ol.style.Style({
            image: new ol.style.Circle({
                radius: 5,
                fill: new ol.style.Fill({
                    color: 'rgba(30, 144, 255, 1)' // 实心蓝色
                }),
                stroke: new ol.style.Stroke({
                    color: 'rgba(30, 144, 255, 0.8)',
                    width: 1
                }),
                displacement: [0, 2] // 向上偏移2像素
            })
        }));
        
        // 创建图层
        this.nodeSelectionLayer = new ol.layer.Vector({
            source: new ol.source.Vector({
                features: [circleFeature]
            }),
            zIndex: 95 // 在节点层之上，扇面之下
        });
        
        this.map.addLayer(this.nodeSelectionLayer);
    }

    /**
     * 清除节点选中效果
     */
    clearNodeSelection() {
        if (this.nodeSelectionLayer) {
            this.map.removeLayer(this.nodeSelectionLayer);
            this.nodeSelectionLayer = null;
        }
    }

    /**
     * 绘制小区扇面（Cell Sector）- 白色固定大小
     * @param {Object} nodeData - 节点数据，包含经纬度、方位角等信息
     * @param {Number} nodeData.lat - 纬度
     * @param {Number} nodeData.lon - 经度
     * @param {Number} nodeData.direct - 水平方位角（度）
     * @param {Number} nodeData.angles - 扇面角度（默认115度）
     * @returns {ol.layer.Vector} 扇面图层
     */
    drawCellSector(nodeData) {
        let {
            lat,
            lon,
            direct = 0,
            angles = 115,
            code
        } = nodeData;
        
        // 确保direct是数字类型
        direct = parseFloat(direct) || 0;

        if (!lat || !lon) {
            console.warn('无效的节点坐标:', { lat, lon });
            return null;
        }

        // 移除旧的小区扇面图层
        if (this.cellSectorLayer) {
            this.map.removeLayer(this.cellSectorLayer);
        }

        // 绘制节点选中效果（实心蓝色圆圈）
        this.drawNodeSelection(nodeData);

        const center = ol.proj.fromLonLat([parseFloat(lon), parseFloat(lat)]);
        
        // 创建点Feature（用于定位）
        const pointFeature = new ol.Feature({
            geometry: new ol.geom.Point(center),
            featureType: 'cellSector',
            nodeCode: code,
            nodeData: nodeData
        });
        
        // 使用Canvas绘制固定像素大小的扇面图标
        const outerRadius = 25; // 外半径像素（25）
        const innerRadius = 8;  // 内半径像素（8）
        const canvas = document.createElement('canvas');
        const size = outerRadius * 2 + 10;
        canvas.width = size;
        canvas.height = size;
        const ctx = canvas.getContext('2d');
        
        // 计算扇面参数
        const centerX = size / 2;
        const centerY = size / 2;
        
        // 先旋转Canvas坐标系，使扇面指向指定方向
        ctx.translate(centerX, centerY);
        ctx.rotate(direct * Math.PI / 180); // 旋转到目标方向
        ctx.translate(-centerX, -centerY);
        
        // 调整角度使其与蓝色扇面方向一致
        // Canvas 的0度是右侧(3点钟方向)，需要调整为正北(12点钟方向)
        // 先减90度让0度指向北，然后再应用扇面角度
        const startAngle = (-90 - angles / 2) * Math.PI / 180;
        const endAngle = (-90 + angles / 2) * Math.PI / 180;
        
        // 绘制扇面
        ctx.beginPath();
        ctx.arc(centerX, centerY, outerRadius, startAngle, endAngle);
        ctx.arc(centerX, centerY, innerRadius, endAngle, startAngle, true);
        ctx.closePath();
        
        // 填充白色
        ctx.fillStyle = 'rgba(255, 255, 255, 0.8)';
        ctx.fill();
        
        // 描边蓝色
        ctx.strokeStyle = 'rgba(255, 255, 255, 0.5)';
        ctx.lineWidth = 1;
        ctx.stroke();
        
        // 设置样式（使用canvas图标）
        pointFeature.setStyle(new ol.style.Style({
            image: new ol.style.Icon({
                img: canvas,
                imgSize: [size, size],
                rotation: 0, // Canvas已经在绘制时旋转了，这里不需要再旋转
                rotateWithView: false // 不随地图旋转
            })
        }));

        // 创建图层
        this.cellSectorLayer = new ol.layer.Vector({
            source: new ol.source.Vector({
                features: [pointFeature]
            }),
            zIndex: 100 // 确保在节点层上方
        });

        this.map.addLayer(this.cellSectorLayer);

        // 保存当前小区扇面数据，用于后续绘制信号覆盖扇面
        this.currentCellSector = {
            nodeData: nodeData,
            feature: pointFeature
        };

        return this.cellSectorLayer;
    }

    /**
     * 绘制信号覆盖扇面（Signal Coverage Sector）- 蓝色渐变大范围
     * @param {Object} nodeData - 节点数据
     * @returns {ol.layer.Vector} 信号覆盖扇面图层
     */
    drawSignalCoverageSector(nodeData) {
        let {
            lat,
            lon,
            direct = 0,
            radius,
            minRadius,
            angles = 115,
            code
        } = nodeData;
        
        // 确保direct是数字类型
        direct = parseFloat(direct) || 0;

        if (!lat || !lon) {
            console.warn('无效的节点坐标:', { lat, lon });
            return null;
        }

        // 如果radius或minRadius无效，重新计算
        if (!radius || radius <= 0 || !minRadius || minRadius <= 0) {
            const calculated = this.calculateSectorRadius(nodeData);
            radius = calculated.radius;
            minRadius = calculated.minRadius;
        }

        // 确保蓝色扇面的视觉大小（屏幕像素）始终大于白色扇面
        // 白色扇面固定大小为20px，蓝色扇面应该是80px（4倍）
        const resolution = this.map.getView().getResolution(); // 获取当前地图分辨率（米/像素）
        
        const blueRadiusPixels = 80;  // 蓝色扇面外半径（像素） = 白色的4倍
        const blueMinRadiusPixels = 40; // 蓝色扇面内半径（像素） = 白色的2倍
        
        // 关键：直接使用像素倍数计算地理距离，不设置固定的最小值
        // 这样可以确保在任何缩放级别下，蓝色扇面在屏幕上都是白色的4倍大
        const targetRadius = blueRadiusPixels * resolution;
        const targetMinRadius = blueMinRadiusPixels * resolution;
        
        // 但为了避免扇面过大（特别是在zoom < 11时），设置合理的最大值
        const MAX_DISPLAY_RADIUS = 50000;     // 最大外半径 50km（增大到可以在小缩放级别看到）
        const MAX_DISPLAY_MIN_RADIUS = 25000;  // 最大内半径 25km
        
        // 使用目标值（确保视觉大小）和原始计算值中的较大者
        radius = Math.max(radius, targetRadius);
        minRadius = Math.max(minRadius, targetMinRadius);
        
        // 应用最大值限制
        if (radius > MAX_DISPLAY_RADIUS) {
            radius = MAX_DISPLAY_RADIUS;
        }
        if (minRadius > MAX_DISPLAY_MIN_RADIUS) {
            minRadius = MAX_DISPLAY_MIN_RADIUS;
        }

        // 移除旧的信号覆盖扇面图层（不移除白色小区扇面）
        if (this.signalCoverageSectorLayer) {
            this.map.removeLayer(this.signalCoverageSectorLayer);
        }

        const center = ol.proj.fromLonLat([parseFloat(lon), parseFloat(lat)]);

        // 创建渐变图层（多个扇形叠加实现从内到外的渐变效果）
        const features = [];
        const gradientLayers = 12; // 增加层数，让过渡更平滑
        
        for (let i = 0; i < gradientLayers; i++) {
            const innerR = minRadius + (radius - minRadius) * (i / gradientLayers);
            const outerR = minRadius + (radius - minRadius) * ((i + 1) / gradientLayers);
            
            // 使用指数衰减：从内到外透明度快速降低，最外层几乎完全透明
            // 内层0.7，最外层接近0
            const ratio = i / gradientLayers;
            const opacity = 0.7 * Math.pow(1 - ratio, 1.5); // 指数1.5让衰减更快
            
            // 创建该层的扇形几何
            const layerGeometry = this.createSectorGeometry(center, outerR, innerR, direct, angles);
            
            // 创建该层的Feature
            const layerFeature = new ol.Feature({
                geometry: layerGeometry,
                featureType: 'signalCoverage',
                nodeCode: code,
                nodeData: nodeData,
                layerIndex: i
            });
            
            // 创建该层的样式（蓝色）
            const layerStyle = new ol.style.Style({
                fill: new ol.style.Fill({
                    color: `rgba(30, 144, 255, ${opacity})` // 蓝色 (DodgerBlue)，透明度递减
                })
                // 不显示边框，让渐变更自然
            });
            
            layerFeature.setStyle(layerStyle);
            features.push(layerFeature);
        }

        // 创建信号覆盖扇面图层
        this.signalCoverageSectorLayer = new ol.layer.Vector({
            source: new ol.source.Vector({
                features: features
            }),
            zIndex: 99 // 在白色小区扇面下方
        });

        this.map.addLayer(this.signalCoverageSectorLayer);

        // 保存当前信号覆盖扇面数据
        this.currentSignalCoverage = {
            nodeData: nodeData,
            allFeatures: features
        };

        return this.signalCoverageSectorLayer;
    }

    /**
     * 为扇面添加鼠标悬浮效果
     */
    addSectorHoverEffect() {
        let hoveredFeature = null;
        
        this.map.on('pointermove', (evt) => {
            if (evt.dragging) return;
            
            const pixel = this.map.getEventPixel(evt.originalEvent);
            const feature = this.map.forEachFeatureAtPixel(pixel, (feature) => {
                const featureType = feature.get('featureType');
                if (featureType === 'cellSector') {
                    return feature;
                }
            });
            
            // 恢复之前高亮的扇面
            if (hoveredFeature && hoveredFeature !== feature) {
                const originalStyle = new ol.style.Style({
                    fill: new ol.style.Fill({
                        color: 'rgba(255, 200, 0, 0.3)'
                    }),
                    stroke: new ol.style.Stroke({
                        color: '#FFC800',
                        width: 2
                    })
                });
                hoveredFeature.setStyle(originalStyle);
            }
            
            // 高亮当前扇面
            if (feature) {
                const hoverStyle = new ol.style.Style({
                    fill: new ol.style.Fill({
                        color: 'rgba(255, 200, 0, 0.5)' // 更明显的黄色
                    }),
                    stroke: new ol.style.Stroke({
                        color: '#FFB800',
                        width: 3
                    })
                });
                feature.setStyle(hoverStyle);
                this.map.getTargetElement().style.cursor = 'pointer';
            } else {
                this.map.getTargetElement().style.cursor = '';
            }
            
            hoveredFeature = feature;
        });
    }

    /**
     * 创建扇面几何图形
     * @param {Array} center - 中心点坐标 [x, y]（投影坐标）
     * @param {Number} outerRadius - 外半径（米）
     * @param {Number} innerRadius - 内半径（米）
     * @param {Number} azimuth - 方位角（度，正北为0，顺时针）
     * @param {Number} angle - 扇面角度（度）
     * @returns {ol.geom.Polygon} 扇面多边形
     */
    createSectorGeometry(center, outerRadius, innerRadius, azimuth, angle) {
        const points = [];
        const segments = 50; // 弧线分段数，越大越平滑
        
        // 转换角度：方位角0度为正北，顺时针旋转
        // OpenLayers 中，0度为正东，逆时针旋转
        // 转换公式：OpenLayers角度 = 90 - 地理方位角
        // 扇面中心应该指向 (90 - azimuth) 度
        const centerAngle = 90 - azimuth;
        const startAngle = this.toRadians(centerAngle - angle / 2);
        const endAngle = this.toRadians(centerAngle + angle / 2);

        // 绘制外弧（从起始角到结束角）
        for (let i = 0; i <= segments; i++) {
            const currentAngle = startAngle + (endAngle - startAngle) * (i / segments);
            const x = center[0] + outerRadius * Math.cos(currentAngle);
            const y = center[1] + outerRadius * Math.sin(currentAngle);
            points.push([x, y]);
        }

        // 绘制内弧（从结束角到起始角，反向）
        if (innerRadius > 0) {
            for (let i = segments; i >= 0; i--) {
                const currentAngle = startAngle + (endAngle - startAngle) * (i / segments);
                const x = center[0] + innerRadius * Math.cos(currentAngle);
                const y = center[1] + innerRadius * Math.sin(currentAngle);
                points.push([x, y]);
            }
        } else {
            // 如果内半径为0，添加中心点
            points.push(center);
        }

        // 闭合路径
        points.push(points[0]);

        return new ol.geom.Polygon([points]);
    }

    /**
     * 角度转弧度
     */
    toRadians(degrees) {
        return degrees * Math.PI / 180;
    }

    /**
     * 清除小区扇面（白色固定大小）
     */
    clearCellSector() {
        if (this.cellSectorLayer) {
            this.map.removeLayer(this.cellSectorLayer);
            this.cellSectorLayer = null;
        }
        this.currentCellSector = null;
    }

    /**
     * 清除信号覆盖扇面（蓝色渐变）
     */
    clearSignalCoverageSector() {
        if (this.signalCoverageSectorLayer) {
            this.map.removeLayer(this.signalCoverageSectorLayer);
            this.signalCoverageSectorLayer = null;
        }
        this.currentSignalCoverage = null;
    }

    /**
     * 清除所有扇面和节点选中效果
     */
    clearAllSectors() {
        this.clearNodeSelection();
        this.clearCellSector();
        this.clearSignalCoverageSector();
    }

    /**
     * ==============================
     * 电子围栏相关方法
     * ==============================
     */

    /**
     * 初始化电子围栏图层
     */
    initFenceLayer() {
        if (!this.map) {
            console.error('Map not initialized. Call initMap() first.');
            return;
        }
        
        // 如果图层已存在且已添加到地图，直接返回
        if (this.fenceSource && this.fenceLayer) {
            // 检查图层是否还在地图上
            const layers = this.map.getLayers().getArray();
            if (layers.indexOf(this.fenceLayer) !== -1) {
                return; // 图层已正确初始化
            }
        }
        
        // 创建或重新创建数据源
        if (!this.fenceSource) {
            this.fenceSource = new ol.source.Vector();
        }
        
        // 创建或重新创建图层
        if (!this.fenceLayer) {
            this.fenceLayer = new ol.layer.Vector({
                source: this.fenceSource,
                style: new ol.style.Style({
                    fill: new ol.style.Fill({
                        color: 'rgba(255, 165, 0, 0.15)' // 浅橘色背景 (15% 透明度)
                    }),
                    stroke: new ol.style.Stroke({
                        color: '#FF8C00', // 橘色边框 (深橙色)
                        width: 2,
                        lineDash: [5, 5]
                    }),
                    image: new ol.style.Circle({
                        radius: 5,
                        fill: new ol.style.Fill({
                            color: '#FF8C00' // 橘色圆点
                        })
                    })
                }),
                zIndex: 5 // 设置在连线(1)之上，但在节点(10)之下，避免遮挡节点交互
            });
        }
        
        // 确保图层添加到地图
        const layers = this.map.getLayers().getArray();
        if (layers.indexOf(this.fenceLayer) === -1) {
            this.map.addLayer(this.fenceLayer);
        }
    }

    /**
     * 创建绘制交互
     * @param {string} drawType - 绘制类型: 'Polygon' 或 'Circle'
     * @param {Function} onDrawEnd - 绘制完成回调函数
     * @returns {ol.interaction.Draw} 绘制交互对象
     */
    createDrawInteraction(drawType, onDrawEnd) {
        // 确保围栏图层已初始化
        this.initFenceLayer();
        
        if (!this.fenceSource) {
            console.error('Fence layer initialization failed');
            return null;
        }

        // 移除已存在的绘制交互
        if (this.currentDrawInteraction) {
            this.map.removeInteraction(this.currentDrawInteraction);
            this.currentDrawInteraction = null;
        }

        // 创建新的绘制交互
        const draw = new ol.interaction.Draw({
            source: this.fenceSource,
            type: drawType,/*
            style: new ol.style.Style({
                fill: new ol.style.Fill({
                    color: 'rgba(255, 165, 0, 0.2)' // 浅橘色背景 (20% 透明度，绘制时稍深)
                }),
                stroke: new ol.style.Stroke({
                    color: '#FF8C00', // 橘色边框
                    width: 3,
                    lineDash: [10, 10]
                }),
                image: new ol.style.Circle({
                    radius: 5,
                    fill: new ol.style.Fill({
                        color: '#FF8C00' // 橘色圆点
                    })
                })
            })*/
        });

        // 绘制完成事件
        draw.on('drawend', (event) => {
            const feature = event.feature;
            const geometry = feature.getGeometry();
            
            // 提取围栏参数
            const fenceParams = this.extractFenceParams(geometry, drawType);
            
            console.log('Draw end event fired, fenceParams:', fenceParams);
            
            if (onDrawEnd) {
                onDrawEnd(fenceParams, feature);
            }

            // 移除绘制交互
            setTimeout(() => {
                this.map.removeInteraction(draw);
                this.currentDrawInteraction = null;
            }, 100);
        });

        this.currentDrawInteraction = draw;
        this.map.addInteraction(draw);

        return draw;
    }

    /**
     * 提取围栏参数
     * @param {ol.geom.Geometry} geometry - OpenLayers几何对象
     * @param {string} type - 几何类型
     * @returns {Object} 围栏参数
     */
    extractFenceParams(geometry, type) {
        const params = {
            type: type,
            timestamp: new Date().getTime()
        };

        if (type === 'Circle') {
            // 圆形围栏
            const center = geometry.getCenter();
            const radius = geometry.getRadius();
            const centerLonLat = ol.proj.toLonLat(center);

            params.center = {
                lon: centerLonLat[0],
                lat: centerLonLat[1]
            };
            params.radius = radius; // 单位：米（投影坐标系）
            params.radiusInMeters = this.calculateRadiusInMeters(center, radius);
        } else if (type === 'Polygon') {
            // 多边形围栏
            const coordinates = geometry.getCoordinates()[0];
            const lonLatCoords = coordinates.map(coord => {
                const lonLat = ol.proj.toLonLat(coord);
                return {
                    lon: lonLat[0],
                    lat: lonLat[1]
                };
            });

            params.coordinates = lonLatCoords;
        }

        return params;
    }

    /**
     * 计算实际距离（米）
     * @param {Array} center - 中心点坐标
     * @param {number} radius - 投影坐标系中的半径
     * @returns {number} 实际距离（米）
     */
    calculateRadiusInMeters(center, radius) {
        // 创建一个距离中心点radius距离的点
        const edgePoint = [center[0] + radius, center[1]];
        
        // 转换为经纬度
        const centerLonLat = ol.proj.toLonLat(center);
        const edgeLonLat = ol.proj.toLonLat(edgePoint);
        
        // 使用Haversine公式计算实际距离
        return this.haversineDistance(
            centerLonLat[1], centerLonLat[0],
            edgeLonLat[1], edgeLonLat[0]
        );
    }

    /**
     * Haversine公式计算两点间距离
     * @param {number} lat1 - 纬度1
     * @param {number} lon1 - 经度1
     * @param {number} lat2 - 纬度2
     * @param {number} lon2 - 经度2
     * @returns {number} 距离（米）
     */
    haversineDistance(lat1, lon1, lat2, lon2) {
        const R = 6371000; // 地球半径（米）
        const dLat = this.toRadians(lat2 - lat1);
        const dLon = this.toRadians(lon2 - lon1);
        
        const a = Math.sin(dLat / 2) * Math.sin(dLat / 2) +
                  Math.cos(this.toRadians(lat1)) * Math.cos(this.toRadians(lat2)) *
                  Math.sin(dLon / 2) * Math.sin(dLon / 2);
        
        const c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a));
        return R * c;
    }

    /**
     * 根据参数绘制围栏（回显）
     * @param {Object} fenceParams - 围栏参数
     * @returns {ol.Feature} 创建的要素
     */
    drawFenceFromParams(fenceParams) {
        this.initFenceLayer();

        let geometry;

        // 多边形围栏
        const coordinates = fenceParams.coordinates.map(coord => 
            ol.proj.fromLonLat([coord.lon, coord.lat])
        );
        geometry = new ol.geom.Polygon([coordinates]);
        

        if (geometry) {
            const feature = new ol.Feature({
                geometry: geometry,
                fenceParams: fenceParams,
                fenceId: fenceParams.id // 添加 fenceId 属性以便定位功能查找
            });

            this.fenceSource.addFeature(feature);
            return feature;
        }

        return null;
    }

    /**
     * 根据中心点和距离计算边缘点
     * @param {number} lat - 纬度
     * @param {number} lon - 经度
     * @param {number} distance - 距离（米）
     * @returns {Object} 边缘点 {lon, lat}
     */
    calculateEdgePoint(lat, lon, distance) {
        const R = 6371000; // 地球半径（米）
        const bearing = 90; // 向东方向
        
        const lat1 = this.toRadians(lat);
        const lon1 = this.toRadians(lon);
        const angularDistance = distance / R;
        const bearingRad = this.toRadians(bearing);
        
        const lat2 = Math.asin(
            Math.sin(lat1) * Math.cos(angularDistance) +
            Math.cos(lat1) * Math.sin(angularDistance) * Math.cos(bearingRad)
        );
        
        const lon2 = lon1 + Math.atan2(
            Math.sin(bearingRad) * Math.sin(angularDistance) * Math.cos(lat1),
            Math.cos(angularDistance) - Math.sin(lat1) * Math.sin(lat2)
        );
        
        return {
            lat: lat2 * 180 / Math.PI,
            lon: lon2 * 180 / Math.PI
        };
    }

    /**
     * 清除所有围栏
     */
    clearAllFences() {
        if (this.fenceSource) {
            this.fenceSource.clear();
        }
    }

    /**
     * 根据 ID 移除单个围栏
     * @param {String} fenceId - 围栏 ID
     * @returns {Boolean} 是否成功移除
     */
    removeFenceById(fenceId) {
        if (!this.fenceSource || !fenceId) {
            return false;
        }

        const features = this.fenceSource.getFeatures();
        let removed = false;

        features.forEach(feature => {
            const fenceParams = feature.get('fenceParams');
            if (fenceParams && fenceParams.id === fenceId) {
                this.fenceSource.removeFeature(feature);
                removed = true;
            }
        });

        return removed;
    }

    /**
     * 添加围栏点击监听器
     * @param {Function} callback - 点击围栏时的回调函数，参数为围栏参数对象
     */
    addFenceClickListener(callback) {
        if (!this.map || typeof callback !== 'function') {
            return;
        }

        // 如果已经有监听器，先移除旧的
        if (this.fenceClickListener) {
            this.map.un('click', this.fenceClickListener);
            this.fenceClickListener = null;
        }

        // 创建新的监听器函数
        this.fenceClickListener = (evt) => {
            // 首先检查是否点击了节点或其他高优先级要素
            let hasNodeOrHighPriorityFeature = false;
            let clickedFenceFeature = null;
            
            this.map.forEachFeatureAtPixel(evt.pixel, (feature, layer) => {
                // 检查是否是节点图层（包括主图层、展开图层）
                if (layer && (layer === this.vectorLayer || 
                    (this.expandedCluster && layer === this.expandedCluster.expandLayer))) {
                    hasNodeOrHighPriorityFeature = true;
                    return true; // 停止遍历，优先处理节点
                }
                
                // 检查是否是小区扇面或信号覆盖扇面
                const featureType = feature.get('featureType');
                if (featureType === 'cellSector' || featureType === 'signalCoverage') {
                    hasNodeOrHighPriorityFeature = true;
                    return true; // 停止遍历，优先处理扇面
                }
                
                // 检查是否是围栏图层
                if (layer && layer === this.fenceLayer && !clickedFenceFeature) {
                    clickedFenceFeature = feature;
                    // 不返回true，继续遍历以确保没有节点在围栏上方
                }
            });

            // 只有在没有点击节点或其他高优先级要素时，才处理围栏点击
            if (!hasNodeOrHighPriorityFeature && clickedFenceFeature) {
                const fenceParams = clickedFenceFeature.get('fenceParams');
                if (fenceParams) {
                    callback(fenceParams);
                }
            }
        };

        // 添加新的监听器
        this.map.on('click', this.fenceClickListener);
    }

    /**
     * 移除绘制交互
     */
    removeDrawInteraction() {
        if (this.currentDrawInteraction) {
            this.map.removeInteraction(this.currentDrawInteraction);
            this.currentDrawInteraction = null;
        }
    }

    /**
     * 获取所有围栏参数
     * @returns {Array} 围栏参数数组
     */
    getAllFenceParams() {
        if (!this.fenceSource) {
            return [];
        }

        const features = this.fenceSource.getFeatures();
        return features.map(feature => feature.get('fenceParams')).filter(params => params);
    }

    /**
     * 启用围栏修改功能
     * @param {Function} onModifyEnd - 修改完成回调
     */
    enableFenceModify(onModifyEnd) {
        this.initFenceLayer();

        if (this.fenceModifyInteraction) {
            this.map.removeInteraction(this.fenceModifyInteraction);
        }

        this.fenceModifyInteraction = new ol.interaction.Modify({
            source: this.fenceSource
        });

        if (onModifyEnd) {
            this.fenceModifyInteraction.on('modifyend', (event) => {
                const features = event.features.getArray();
                const updatedParams = features.map(feature => {
                    const geometry = feature.getGeometry();
                    const type = geometry.getType() === 'Circle' ? 'Circle' : 'Polygon';
                    return this.extractFenceParams(geometry, type);
                });
                onModifyEnd(updatedParams);
            });
        }

        this.map.addInteraction(this.fenceModifyInteraction);
    }

    /**
     * 禁用围栏修改功能
     */
    disableFenceModify() {
        if (this.fenceModifyInteraction) {
            this.map.removeInteraction(this.fenceModifyInteraction);
            this.fenceModifyInteraction = null;
        }
    }

    /**
     * 检测围栏内的设备
     * @param {Object} geometry - OL geometry 对象（Polygon 或 Circle）
     * @param {Array} allNodes - 所有节点数据数组
     * @returns {Array} 围栏内的设备列表
     */
    getDevicesInFence(geometry, allNodes) {
        if (!geometry || !allNodes || allNodes.length === 0) {
            return [];
        }

        const devicesInFence = [];
        
        allNodes.forEach(node => {
            // 确保节点有经纬度信息
            if (!node.lon || !node.lat) {
                return;
            }

            try {
                // 创建节点的坐标点（EPSG:4326）
                const nodeCoord = [parseFloat(node.lon), parseFloat(node.lat)];
                
                // 转换为地图坐标系 (EPSG:3857)
                const transformedCoord = ol.proj.transform(nodeCoord, 'EPSG:4326', 'EPSG:3857');
                
                // 检查点是否在几何图形内
                let isInside = false;
                
                if (geometry.getType() === 'Polygon') {
                    // 多边形判断
                    isInside = geometry.intersectsCoordinate(transformedCoord);
                } else if (geometry.getType() === 'Circle') {
                    // 圆形判断
                    const center = geometry.getCenter();
                    const radius = geometry.getRadius();
                    const distance = Math.sqrt(
                        Math.pow(transformedCoord[0] - center[0], 2) +
                        Math.pow(transformedCoord[1] - center[1], 2)
                    );
                    isInside = distance <= radius;
                }
                
                if (isInside) {
                    devicesInFence.push({
                        serialNumber: node.sn || node.serialNumber || 'N/A',
                        cellName: node.name || node.cellName || 'N/A',
                        type: node.deviceType || node.type || this.getDeviceType(node),
                        code: node.code,
                        fullData: node // 保存完整节点数据以备后用
                    });
                }
            } catch (e) {
                console.error('检测设备时出错:', e, node);
            }
        });

        return devicesInFence;
    }

    /**
     * 根据节点数据推断设备类型
     * @param {Object} node - 节点数据
     * @returns {String} 设备类型
     */
    getDeviceType(node) {
        // 根据节点属性推断类型
        if (node.isGnb || node.type === 'gNB') return 'gNB';
        if (node.isCpe || node.type === 'CPE') return 'CPE';
        if (node.isWcg || node.type === 'WCG') return 'WCG';
        return 'eNB'; // 默认类型
    }

    /**
     * 更新节点为 KPI 模式样式（批量更新）
     * @param {Object} nodeStyles - 节点样式映射 {nodeCode: {color: '#FF5B45', value: 100}}
     * @param {Boolean} showValue - 是否显示值
     */
    updateNodeStyles(nodeStyles, showValue) {
        if (!this.vectorSource) {
            console.warn('Vector source not initialized');
            return;
        }

        // 设置 KPI 模式标志
        this.kpiMode = true;
        this.kpiStyles = nodeStyles;
        this.kpiShowValue = showValue;

        // 强制刷新图层，触发 getFeatureStyle 重新计算
        if (this.vectorLayer) {
            this.vectorLayer.changed();
        }

        // 如果有展开的重叠节点，也需要刷新展开图层
        if (this.expandedCluster && this.expandedCluster.expandLayer) {
            const expandLayer = this.expandedCluster.expandLayer;
            const expandSource = expandLayer.getSource();
            if (expandSource) {
                // 触发展开图层重新渲染，使 getFeatureStyle 重新计算展开节点的样式
                expandSource.changed();
            }
        }

        console.log('KPI Mode: Updated styles for', Object.keys(nodeStyles).length, 'nodes');
    }

    /**
     * 更新单个节点的 KPI 样式
     * @param {String} nodeCode - 节点代码
     * @param {Object} style - 样式对象 {color: '#FF5B45', value: 100}
     */
    updateSingleNodeStyle(nodeCode, style) {
        if (!this.kpiStyles) {
            this.kpiStyles = {};
        }

        this.kpiStyles[nodeCode] = style;

        // 强制刷新图层
        if (this.vectorLayer) {
            this.vectorLayer.changed();
        }

        // 如果有展开的重叠节点，也需要刷新展开图层
        if (this.expandedCluster && this.expandedCluster.expandLayer) {
            const expandLayer = this.expandedCluster.expandLayer;
            const expandSource = expandLayer.getSource();
            if (expandSource) {
                expandSource.changed();
            }
        }
    }

    /**
     * 清除 KPI 模式，恢复正常渲染
     */
    clearKpiMode() {
        this.kpiMode = false;
        this.kpiStyles = null;
        this.kpiShowValue = false;

        // 强制刷新图层，恢复正常样式
        if (this.vectorLayer) {
            this.vectorLayer.changed();
        }

        // 如果有展开的重叠节点，也需要刷新展开图层
        if (this.expandedCluster && this.expandedCluster.expandLayer) {
            const expandLayer = this.expandedCluster.expandLayer;
            const expandSource = expandLayer.getSource();
            if (expandSource) {
                expandSource.changed();
            }
        }

        console.log('KPI Mode: Cleared');
    }

    /**
     * 定位到指定围栏
     * @param {String} fenceId - 围栏ID
     */
    locateToFence(fenceId) {
        if (!this.map || !this.fenceSource) {
            console.error('Map or fence source not initialized');
            return;
        }

        // 在围栏数据源中查找指定ID的围栏
        const features = this.fenceSource.getFeatures();
        console.log('所有围栏 features:', features.length);
        
        // 首先尝试通过 fenceId 属性查找
        let targetFeature = features.find(feature => feature.get('fenceId') === fenceId);
        
        // 如果没找到，尝试通过 fenceParams.id 查找（兼容旧数据）
        if (!targetFeature) {
            targetFeature = features.find(feature => {
                const fenceParams = feature.get('fenceParams');
                return fenceParams && fenceParams.id === fenceId;
            });
        }

        if (!targetFeature) {
            console.warn('Fence not found with ID:', fenceId);
            console.log('Available fence IDs:', features.map(f => f.get('fenceId') || (f.get('fenceParams') && f.get('fenceParams').id)));
            return;
        }

        // 获取围栏的范围
        const geometry = targetFeature.getGeometry();
        const extent = geometry.getExtent();

        // 添加一些边距，使围栏不会填满整个视图
        const buffer = (extent[2] - extent[0]) * 0.2; // 20% 的边距
        const bufferedExtent = [
            extent[0] - buffer,
            extent[1] - buffer,
            extent[2] + buffer,
            extent[3] + buffer
        ];

        // 使用动画效果定位到围栏
        this.map.getView().fit(bufferedExtent, {
            duration: 1000, // 动画持续时间（毫秒）
            padding: [50, 50, 50, 50], // 视图边距
            maxZoom: 16 // 最大缩放级别，避免过度放大
        });

        console.log('Located to fence:', fenceId);
    }
}

//导出到全局
window.OLTopoHelper = OLTopoHelper;
