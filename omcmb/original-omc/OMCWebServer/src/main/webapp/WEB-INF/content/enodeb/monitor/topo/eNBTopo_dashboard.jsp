<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<link rel="stylesheet" href="${ctx}/js/topo/leaflet.css" type="text/css" media="screen" />
<link rel="stylesheet" href="${ctx}/js/topo/dvf.css" type="text/css" media="screen"/>
<link rel="stylesheet" href="${ctx}/js/topo/node.css" type="text/css" media="screen"/>
<style>
	.width-mini .el-input.el-input--small {
		width: 78% !important;
	}
	.width-mini .advanceQuery {
		margin-left: 0px;
	}
	.mmeDetails , .syncNameInfo{
		position:absolute;
		background:white;
		padding:10px 20px;
		box-shadow:4px 4px 19px 0 rgba(0,0,0,0.15);
		border:1px solid #d1ecf5;
		display:none;
		left:0;
		bottom: 45px;
		z-index:99999;
	}
	.device-slider {
		padding: 15px 10px;
		position: absolute; 
		top: 40px;
		left: -800px;
		width: 450px;
		height: 70%; 
		max-height: 500px;
		background-color: #fff;
		box-shadow: 2px 3px 8px #cdc6c6;
		transition: left 0.5s ease;
	}
	.device-slider.show {
		left: 241px;
		z-index: 100;
	}
	.ellipsis-txt {
		display: inline-block;
		overflow: hidden;
		text-overflow: ellipsis;
	}
	.absolute-ctn {
		position: absolute;
		top: 0;
		right: 0;
		bottom: 0;
		left: 0;
		z-index: 101;
	}
	.flex-align-center {
		display: flex;
		align-items: baseline;
	}
	.form-item-bottom-15 .el-form-item {
		margin-bottom: 15px;
	}
	.form-item-bottom-15 .el-form-item__label {
		line-height: 22px;
	}

	.leaflet-popup-content .node-info, .leaflet-popup-content .node-gps, 
	.leaflet-popup-content.gps .node-info, .leaflet-popup-content.view .node-gps {
		opacity: 1;
		height: auto;
		width: 200px;
	}
	.node-label {
		display: inline-block;
		width: 90px;
		text-align: right;
		color: #B0B0B0;
		white-space: nowrap
	}
	.node-ul {
		padding-top: 5px;
	}
	.node-ul li {
		padding: 2px;
	}
	.alarm-info {
		right: -45px;
	}
	.node-item-leaf .el-icon-topo-cpe::before, .el-icon-topo-cpe.online::before {
		color: #3EA20D;
	}
	.node-item-leaf.offline .el-icon-topo-cpe::before,.node-item-leaf.offline .el-icon-topo-enb::before,
	.el-icon.offline::before {
		color: #666;
	}
	.topo-legend-cls {
		padding: 80px 15px 10px 15px;
		position: absolute; 
		right: 15px; 
		top: 10px;
		background-color: #fff;
		z-index: 10;
		box-shadow: 2px 3px 8px #cdc6c6;
		min-width: 110px;
		border-radius: 5px;
	}
	.topo-legend-cls > div {
		padding: 3px;
	}
	.legend-title {
		top: 0;
		left: 0;
		right: 0;
		position: absolute;
		display: flex;
		align-items: center;
		border-bottom: 2px solid #e9e9e9;
		color: #999;
		font-weight: normal;
	}
	.legend-title > div {
		flex: 1 1 50%;
		text-align: center;
		padding: 5px 0;
	}
	.location-title {
		padding: 5px 0px !important;
		justify-content: center;
		background-color: rgba(51,51,51,0.7);
		font-weight:bold;
		font-size:14px;
		color: #fff;
		border-radius: 5px 5px 0 0;
	}
	.leaflet-popup-content {
		margin: 5px;
	}
	.leaflet-popup-content-wrapper {
		border: none;
		border-radius: 0;
	}
	.leaflet-popup-content .tabsTitle, .device-slider .el-tabs__header {
		background: #fff;
	}
	.leaflet-popup-content .tabsContentDiv {
		border-width: 0px;
		border-top-width: 1px;
	}
	.leaflet-popup-content .tabsTitle {
		color: 363B4E;
	}
	.el-icon-topo-enb.white-bg,.el-icon-topo-cpe.white-bg {
		border: none !important;
	}
	.el-icon-topo-enb.white-bg::after,.el-icon-topo-cpe.white-bg::after{
		content: '';
		padding: 8px;
		background: #fff; 
		position: absolute;
		top: 4px;
		left: 0px;
		border-radius: 8px;
		z-index: -1;
	}
	.node-item-leaf .node-code {
		display: none;
	}
	.node-item-leaf:hover .node-code {
		display: inline;
	}


	.repeated-nodes {
		display: inline-block;
		width: 60px;
		height: 60px;
		position: absolute;
		top: -15px;
		border: 1px dashed #5c5cf0;
		border-radius: 5px;
		background-color: rgba(178,219,255,0.3);
		z-index: -1;
	}
	.repeated-list {
		display: none;
		padding: 5px 10px;
		position: absolute;
		max-width: 200px;
		min-width: 100px;
		max-height: 180px;
		overflow: auto;
		top: 70px;
		background-color: #fff;
		border-radius: 5px;
		z-index: 100;
	}
	.repeated-item {
		display: block
	}
	.repeated-count {
		padding: 0px 2px;
		position: absolute;
		left: 50px;
		top: 50px;
		display: inline-block;
		height: 13px;
		line-height: 15px;
		min-width: 30px;
		text-align: center;
		background-color: #fff;
		border-radius: 7px;
		border: 1px solid #5c5cf0;
	}
</style>

<div id="topo_container" style="height: 100%;display: flex;border: 1px solid #E9E9E9;background: #fff;position: relative;">
		<!-- 图例 -->
		<div class="topo-legend-cls">
			<div class="legend-title location-title"><%=rb.getString("WeiZhi")%></div>
			<div class="legend-title" style="margin-top: 25px;">
				<div style="border-right: 1px solid #e9e9e9;">
					<%=rb.getString("XiaoZhan")%>
					<div style="font-weight: bold;color: #363b4e;padding-top: 5px;">{{enbNodes.length}}/{{enbNolocation.length + enbNodes.length + repeatedEnbnum}}</div>
				</div>
				<div v-show="isCPEUsable" style="padding-top: 3px;">
					<%=rb.getString("CPE")%>
					<div style="font-weight: bold;color: #363b4e;padding-top: 5px;">{{cpeNodes.length}}/{{cpeNolocation.length + cpeNodes.length + repeatedCpenum}}</div>
				</div>
			</div>
			<div><i class="el-icon el-icon-topo-enb"></i> <%=rb.getString("XiaoZhan")%> -- <%=rb.getString("ZaiXian")%></div>
			<div v-show="isCPEUsable"><i class="el-icon el-icon-topo-cpe online"></i> <%=rb.getString("CPE")%> -- <%=rb.getString("ZaiXian")%></div>
			<div><i class="el-icon el-icon-topo-enb offline"></i> <%=rb.getString("XiaoZhan")%> -- <%=rb.getString("LiXian")%></div>
			<div v-show="isCPEUsable"><i class="el-icon el-icon-topo-cpe offline"></i> <%=rb.getString("CPE")%> -- <%=rb.getString("LiXian")%></div>
		</div>
		<!-- topo 地图 -->
		<div style="flex: auto;overflow: auto;">
			<div id="map" style="width: 100%;min-height: 100%;background: #A1D4E0;"></div>
		</div>
	</div>
</div>
  	
<script type="text/javascript" src="${ctx}/js/topo/leaflet.js"></script>
<script type="text/javascript" src="${ctx}/js/topo/leaflet-dvf.js"></script>
	
<script type="text/javascript">
	function hideRepeatedList() {
		$('#topo_container .repeated-list').fadeOut();
	}
	$('body').off('click',hideRepeatedList).on('click',hideRepeatedList);

	var globMap,groupName = '';
	// 扩展字符串的数据映射能力
	String.prototype.evaluate = function(map,pKey,ptxt){
		if(pKey) pKey += '.';
		else pKey = '';
		
		var txt = ptxt? ptxt:this.toString();
		for(var key in map){
			if(map.hasOwnProperty(key)){
				if(typeof map[key] == 'object'){
					txt = this.evaluate(map[key],pKey+key,txt);
				}else{
					var reg = new RegExp('{\\s*'+pKey+key+'\\s*}','g');
					txt = txt.replace(reg,map[key]);
				}
			}
		}
		if(!ptxt){
			var rg = new RegExp('{\\s*[a-zA-Z0-9\\.]+\\s*}','g');
			txt = txt.replace(rg,'');
		}
		return txt;
	}
    
	var topovm = new Vue({
		el: '#topo_container', 
		data() {
			var vm = this;
			
			return {
				limit: 100, // 过滤数据: 阀值
				links: {}, // enb为key，cpe存于数组中：enbCode:[cpeCode1, cpeCode2, ...]，建立 1:n 关系
				activeName: 'ENB',
				groupURL: '${ctx}/system/deviceGroup/getDeviceGroupList.action',
				cpeNodes: [], // 记录获取的cpe节点
				enbNodes: [], // 记录获取的enb节点
				prevNodes: [], // 缓存的上次绘制节点
				allNodes: [], // cpe、enb节点合集
				enbNolocation: [],
				cpeNolocation: [],
				enbShowList: [],
				cpeShowList: [],
				repeatedMap: {},
				repeatedEnbnum: 0,
				repeatedCpenum: 0
			};
		},
		computed: {
			isCPEUsable() {
				return writableMap['CODE_CPE_MONITOR'] != undefined;
			},
			isEnbUsable() {
				return writableMap['CODE_ENB_MONITOR'] != undefined;
			}
		},
		methods: {
			
			/**
			* 检测经纬度是否都有效
			* @param row{object}: 列表行数据
			**/
			checkLatLng(row) {
				var vm = this,
					lat = row.latitude,
					lng = row.longitude,
					valid = false;

				if(lat !== 'null' && lat !== '' && lat !== null && lng !== 'null' && lng !== '' && lng !== null) {
					valid = true;
				}

				return valid;
			},
			// 取消设置关闭浮层
			cancelSet() {
				document.querySelector('#map').click();
				document.querySelector('#map').click();
			},
			initRepeated(nodes) {
				var vm = this;
					latLonList = nodes.map(function(node){ return node.lat+'-'+node.lon; }),
					repeatedSNList = [];
				
				nodes.map(function(node){
					var code = node.code,
						latlonKey = node.lat+'-'+node.lon;

					if(isMoreThenOne(latlonKey,latLonList)) {
						if(vm.repeatedMap[latlonKey]) {
							vm.repeatedMap[latlonKey].push(code);
							repeatedSNList.push(code);

							if(vm.repeatedMap[latlonKey].length) {
								if(node.type == 'enb') {
									vm.repeatedEnbnum +=1;
								}else {
									vm.repeatedCpenum +=1;
								}
							}
						}else {
							vm.repeatedMap[latlonKey] = [];
						}
					}
				});

				nodes = nodes.filter(function(node){
					return !repeatedSNList.includes(node.code);
				});
				
				return nodes;
			},
			/**
			* 获取设备节点数据绘制topo
			* @param ids{string}: 设备组Id，多个以逗号分隔
			**/
			getTopoInfos() {
				var vm = this,
					url = '${ctx}/cell/topo/getDeviceInfoList.action',
					params = {
						queryType: 'operator',
						device_type: 'ENB',
						operator_code: operator_code
					};
				vm.links = {};
				// 获取enb节点
				axios.post(url,stringify(params)).then(function(res){
					var nodes = res.data.rows;
					// 节点数据格式规范化 -- 应对变化的接口数据格式
					nodes = transformNode(nodes);
					nodes = vm.initRepeated(nodes);
					vm.enbNodes = nodes;

					// 获取cpe节点
					if(writableMap['CODE_CPE_MONITOR'] != undefined){
						params.device_type = 'CPE';
						axios.post(url,stringify(params)).then(function(res){
							var nodes = res.data.rows;

							// 节点数据格式规范化 -- 应对变化的接口数据格式
							nodes = vm.transformCPE(nodes);
							nodes = vm.initRepeated(nodes);
							vm.cpeNodes = nodes;
							vm.initMap();
						}).catch(function(){
							try{
								vm.initMap();
							}catch(e){}
						});
					}else {
						try{
							vm.initMap();
						}catch(e){}
					}
					
				});
			},
			/**
			* 将获取的cpe节点数据转换为规范格式
			* @param nodes{array}: cpe原始数据
			**/
			transformCPE(nodes) {
				var vm = this,
					cpes = [],
					latLons = nodes.map(function(node){ return node.latitude+'-'+node.longitude; });
				
				nodes.map(function(node){
					var lat = node.latitude,
						lon = node.longitude;
					
					if(node.latitude == "null") {
						lat = '';
					}
					if(node.longitude == "null") {
						lon = '';
					}
					// 位置相同的点进行经纬度偏移
					if(isNotNull(lat) && isNotNull(lon) && isMoreThenOne(lat+'-'+lon,latLons) && false){
						var idx = getIndexFromBrothers(node,nodes);
						lon = lon-0 + 0.01*idx + '';
						lat = lat-0 + 0.01*idx + '';
					}
					// 规范格式
					var cpe = {
							code: node.cpe_code,
							lat: lat,
							lon: lon,
							name: node.host_name,
							olat: node.latitude,
							olon: node.longitude,
							groupName: node.group_name,
							type: 'cpe',
							imsi: node.imsi,
							online: node.connection_status? 'on':'off',
							alarmCount: node.alarm_count,
							alarmLevel: node.alarm_serverity,
							ip: node.ipaddress,
							mac: node.macaddress,
							cellCode: node.cpe_code,
							sn: node.serial_number
						};
					
					cpes.push(cpe);

					// 连线关系
					if(node.rela_enb) {
						var key = node.rela_enb,
							cpeCode = node.cpe_code;

						if(vm.links[key]) {
							vm.links[key].push(cpeCode);
						}else {
							vm.links[key] = [cpeCode]
						}
					}
				});

				topovm.cpeNolocation = cpes.filter(function(item){
					return !item.lat || !item.lon;
				});

				cpes = cpes.filter(function(item){
					return item.lat && item.lon;
				});
			
				return cpes;
			},
			/**
			* 获取topo可视边界域
			* @param resObj{object}: 所有节点的统计数据对象
			**/
			getViewBounds(resObj) {

				return [
					[resObj.latmin, resObj.lonmin], // 西南角点
					[resObj.latmax, resObj.lonmax]  // 东北角点
				];
			},
			/**
			* 依据阀值，获取限定的节点
			* @param nodes{array}: 所有要绘制的节点集
			* @param bufEnable{boolean}: 是否开启缓存
			**/
			getLimitedNodes(nodes, bufEnable) {
				var vm = this,
					total = nodes.length,
					limit = this.limit,
					filterNodes = [];

				if(total<=limit) {
					vm.prevNodes = nodes;
					
					return nodes;
				}
				// 排序
				nodes = nodes.sort(function(a, b){
					return Math.pow(a.lat,2) + Math.pow(a.lon, 2) > Math.pow(b.lat,2) + Math.pow(b.lon, 2);
				});
				// 需要缓存上次数据，对比增加和减少
				if(bufEnable === true) {
					var prevNodes = vm.prevNodes, // 历史节点集
						prevKeys = prevNodes.map(function(node){ return node.code;}),
						curKeys = nodes.map(function(node){ return node.code;}), // 当前可视区域节点code集
						mixedNodes = prevNodes.filter(function(node){ return curKeys.includes(node.code); }),
						newNodes = nodes.filter(function(node){ return !prevKeys.includes(node.code); }),
						addNum = limit - mixedNodes.length;//节点交集
					addNum = addNum>0?addNum:0;
					
					filterNodes = mixedNodes.concat(newNodes.slice(0,addNum));
				} else {
					// 根据步长获取节点
					for(var i=0; i<limit; i++) {
						var rate = (total/limit).toFixed(4),
							nodeIndex = Math.floor(rate*i);
						filterNodes.push(nodes[nodeIndex]);
					}
				}
				
				vm.prevNodes = filterNodes;
				
				return filterNodes;
			},
			/**
			* 并入和eNb有连线的CPE节点
			* @param nodes{array}: 所有节点
			**/
			pushLinkNodes(nodes) {
				var vm = this;
				var cpeList = vm.cpeNodes,
					existedKeys = nodes.map(function(item){ return item.code;});
				// 设备是否包含CPE
				//if(!vm.form.device.includes('cpe')) return;

				nodes.map(function(node){
					if(node.type=='enb') {
						var linkcpekeys = vm.links[node.code];
						// 存在连线
						if(linkcpekeys && linkcpekeys.length) {
							linkcpekeys.map(function(cpekey){
								var cpe = cpeList.filter(function(item){
										return item.code == cpekey; // 根据cpe key获取cpe数据
									})[0];
								// 集合不含该节点时，收入集合中
								if(cpe && !existedKeys.includes(cpe.code)) {
									nodes.push(cpe);
								}
							})
						}
					}
				});
			},
			/**
			* 渲染节点和连线图层
			* @param nodes{array}: 所有要绘制的节点集
			* @param map{dom}: 地图实例
			* @param bufEnable{boolean}: 是否开启缓存
			**/
			createEnbLayer(nodes,map,bufEnable) {
				var vm = this;
				
				// 判断可视区域内的节点是否超过阀值
				nodes = this.getLimitedNodes(nodes,bufEnable);
				// 并入与eNb有连线的CPE节点
				vm.pushLinkNodes(nodes);
				// 创建以code为主键的检索队列
	        	var eNodebsLookup = L.GeometryUtils.arrayToMap(nodes, 'code');
				var enbOptions = getEnbLayerOptions();
	            // 渲染eNb和 CPE节点
				var eNodebsLayer = new L.MarkerDataLayer(eNodebsLookup, enbOptions);
				map.addLayer(eNodebsLayer);
				// 渲染eNb和 CPE 的连线
				vm.createLineLayer(nodes,eNodebsLookup,map);
			},
			/**
			* 渲染连线图层
			* @param nodes{array}: 所有要绘制的节点集
			* @param eNbsLookup{array}: 被初始化的enb队列
			* @param map{dom}: 地图实例
			**/
			createLineLayer(nodes,eNbsLookup,map) {
				var vm = this,
					connLines = vm.proccessLines(nodes),
					options = getAllLayerOptions(eNbsLookup);
				
				vm.enbShowList = nodes.filter(function(node){
					return node.type == 'enb';
				});
				vm.cpeShowList = connLines.map(function(item){ return item.tar;});
				nodes.map(function(node){
					if(node.type == 'cpe' && !vm.cpeShowList.includes(node.code)) {
						vm.cpeShowList.push(node.code);
					}
				})

				var allLayer = new L.Graph(connLines, options);
	        	map.addLayer(allLayer);
			},
			/**
			* 地图缩放或拖拽后，对节点重新统计、过滤、渲染
			* @param nodes{array}: 所有要绘制的节点集
			* @param map{dom}: 地图实例
			* @param bufEnable{boolean}: 是否开启缓存
			**/
			filterRender(nodes, map, bufEnable) {
				var vm = this,
					bounds = map.getBounds(),
					sw = bounds._southWest,
					ne = bounds._northEast,
					zoom = map.getZoom(),
					threshold = this.limit;
				// 根据控制状态过滤节点
				// 删除历史节点图层
				map.eachLayer(function(layer){
					// 根据options中的特性，匹配节点和连线的layer，然后删除
					if(layer.options && (layer.options.type == 'enb'||layer.options.fromField == 'enb')) {
						map.removeLayer(layer);
					}
				});

				if(nodes.length > threshold) {
					// 重新渲染layer
					var inviewNodes = nodes.filter(function(item){
							var lat = parseFloat(item.lat),
								lon = parseFloat(item.lon);
							// 判断可视区域内的节点
							return sw.lat<=lat && lat<=ne.lat && sw.lng<=lon && lon<=ne.lng;
						});
					
					vm.createEnbLayer(inviewNodes,map,bufEnable);
				}else {
					vm.createEnbLayer(nodes,map,bufEnable);
				}
			},
			// 初始化topo图
			initMap() {
				var vm = this,
					eNodebs = vm.enbNodes.concat(vm.cpeNodes);
				
				// 自适应窗口设置 -- start
				var map;
				var $map = $('#map');
				var resize = function () {
					$map.height($('.topo-ctn').height() - 20);
		
					if (map) {
						map.invalidateSize();
					}
				};
				// topo随窗口大小自动适应
				$(window).off('resize',resize);
				$(window).on('resize',resize);
				resize();
				
				if(globMap) try{globMap.remove();}catch(e){}
				
				// 处理基站数据格式
				var resObj = vm.proccessNodes(eNodebs);
				// 获取要展示的区域，fitBounds会自动调整合理显示
				var bounds = vm.getViewBounds(resObj);
				// 设置中心位置和放大倍数
				map = L.map('map',{maxZoom: 18,minZoom: 2}).fitBounds(bounds); // .setMaxBounds([[-360,-360],[360,360]])
				globMap = map;
				
				// 关联背景地图资源
				L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png',{
					attribution: ''
				}).addTo(map);
				// 过滤后生成节点图层
				vm.createEnbLayer(resObj.enb, globMap);
				// 节点挂着到全局，方便过滤的时候处理
				vm.allNodes = resObj.enb;

				// 事件绑定
				globMap.on('zoomend',function(ev){ // 缩放结束后重新计算
					vm.filterRender(resObj.enb, globMap);
				});// 事件绑定
				globMap.on('moveend',function(ev){ // 移动结束后重新计算
					vm.filterRender(resObj.enb, globMap,true);
				});
			},
			/**
			* 计算经纬边界值
			* @param nodes{array}: 所有节点
			**/
			proccessNodes(nodes) {
				var cLat = 0, cLon = 0, 
					latMax = -90, lonMax = -180,
					latMin = 90, lonMin = 180;
				
				nodes.map(function(item,index){
					var ilat = parseFloat(item.lat),
						ilon = parseFloat(item.lon);
					if(ilat>latMax && Math.abs(ilat)<=90) latMax = ilat;
					if(ilon>lonMax && Math.abs(ilon)<=180) lonMax = ilon;
					if(ilat<latMin && Math.abs(ilat)<=90) latMin = ilat;
					if(ilon<lonMin && Math.abs(ilon)<=180) lonMin = ilon;
					
				});
				
				// 计算经纬中心点
				cLat = (latMax + latMin)/2;
				cLon = (lonMax + lonMin)/2;

				return {
					enb: nodes,
					latMax: latMax,
					lat: cLat,
					lon: cLon,
					latdis: latMax - latMin,
					londis: lonMax - lonMin,
					latmax: latMax,
					latmin: latMin,
					lonmax: lonMax,
					lonmin: lonMin
				}
			},
			/**
			* 获取连线数据
			* @param nodes{array}: 所有节点
			**/
			proccessLines(nodes) {
				var vm = this,
					lineList = [],
					enblist = nodes.filter(function(item){ return item.type=='enb';}),
					cpelist = nodes.filter(function(item){ return item.type=='cpe';});

				enblist.map(function(node){
					var cpekeys = vm.links[node.code];
					// 存在连线
					if(cpekeys && cpekeys.length) {
						cpekeys.map(function(cpekey){
							var cpe = cpelist.filter(function(item){
									return item.code == cpekey && item.lat!==null && item.lon!==null && item.lat!=="" && item.lon!==""; // 根据cpe key获取cpe数据
								})[0];
							
							if(cpe) {
								var cpeLine = {
										"label": 'CPE',
										"enb": node.code,
										"tar": cpe.code,
										"cnt": "",
										type: 'enb',
										online: node.online,
										active: node.active,
										mmeStatus: node.mmeStatus,
										mmeEnable: node.mmeEnable
									};

								lineList.push(cpeLine);
							}
						})
					}
				});

				return lineList;
			}
		},
		mounted() {
			var vm = this;
			// 获取sas开关状态
			axios.post('${ctx}/cell/topo/getSasEnableStatus.action',stringify({operator_code: operator_code})).then(function(res){
				if(res && res.data) {
					vm.sasEnable = !res.data.success || writableMap.CODE_TOPO !== true;
				}
			});
			
			vm.getTopoInfos();
		}
	});
	/**
	* 刷新topo图 -- 节点数据变动
	* @param obj{object}：信息有变动的节点
	**/
	function refreshTopo(obj) {
		var vm = topovm;
		
		vm.allNodes.map(function(item){
			if(item.cellCode == obj.cellCode) {
				Object.assign(item, obj);
			}
		});
		vm.filterRender(vm.allNodes, globMap)
	}
	/**
	* 判断元素在数组中是否有多个
	* @param item{string}：要判断的值
	* @param list{array}：已有值的集合
	**/
	function isMoreThenOne(item,list){
		return list.indexOf(item) < list.lastIndexOf(item);
	}
	/**
	* 获取节点所在队列的下标
	* @param node{object}：当前节点
	* @param list{array}：兄弟节点集合
	**/
	function getIndexFromBrothers(node,list){
		var brothers = list.filter(function(item){
				return node.gps_latitude !== null && node.gps_longitude !== null && node.gps_latitude+'-'+node.gps_longitude == item.gps_latitude+'-'+item.gps_longitude
			}),index = 0;

		brothers.map(function(item,idx){
			if(node.serial_number == item.serial_number) index = idx;
		});
		return index;
	}
	/**
	* 判空
	* @param val{string}：要判断的值
	**/
	function isNotNull(val){
		var bool = false;
		if(val !== null && val !== '' && val != undefined) bool = true;
		return bool;
	}
	/**
	* enb数据规范化 -- {code: '',lat: '',lon: '',name: '',type: '',online: '',active: '', ... }为必要属性
	* @param nodeList{array}：enb节点原始数据
	**/
	function transformNode(nodeList){
		var latLons = nodeList.map(function(node){ return node.latitude+'-'+node.longitude; });
		
		nodeList = nodeList.map(function(node){
			var enable = node.mme_enable == 1? true:false,
				mmePool_1 = node.mme_pool_1?node.mme_pool_1:'',
				mmePool_2 = node.mme_pool_2?node.mme_pool_2:'',
				serverList = node.s1siglinkserverlist?node.s1siglinkserverlist:'',
				lat = node.latitude,
				lon = node.longitude;
			if(node.latitude == "null") {
				lat = '';
			}
			if(node.longitude == "null") {
				lon = '';
			}
			// 位置相同的点进行经纬度偏移
			if(isNotNull(lat) && isNotNull(lon) && isMoreThenOne(lat+'-'+lon,latLons) && false){
				var idx = getIndexFromBrothers(node,nodeList);
				lon = lon-0 + 0.01*idx + '';
				lat = lat-0 + 0.01*idx + '';
			}
			// 规范属性
			return {
				code: node.serial_number,
				olat: node.latitude,
				olon: node.longitude,
				lat: lat,
				lon: lon,
				name: node.host_name,
				groupName: node.group_name,
				ip: node.cell_ip,
				type: 'enb',
				online: node.connection_status? 'on':'off',
				active: node.op_state? 'yes':'no',
				alarmCount: node.alarm_count,
				alarmLevel: node.alarm_serverity,
				mmeEnable: enable,
				mmeStatus: node.mme_status,
				mmePool1: mmePool_1,
				mmePool2: mmePool_2,
				serverList: serverList,
				cellCode: node.small_cell_code,
				alarm: node.alarm
			};
		});
		// 处理无经纬度的节点
		//proccessNoLocNodes(nodeList);

		topovm.enbNolocation = nodeList.filter(function(item){
			return !item.lat || !item.lon;
		});
		// 过滤无经纬度的节点
		nodeList = nodeList.filter(function(item){
			return item.lat && item.lon;
		});

		return nodeList;
	}
	/**
	* 高亮显示选中节点
	* @param code{string}：节点code
	**/
	function highlightNode(code){
		$('.node-code').each(function(idx,item){
			var $dom = $(item);
			if($dom.text() == code) $dom.addClass('selected');
			else $dom.removeClass('selected')
		});
	}
	
	function showRepeatedList(dom,event) {
		$('.repeated-list').fadeOut();
		$('.repeated-list',dom.parentNode).fadeIn();
		event.stopPropagation();
	}
	// 生成enb节点的图层配置项
	function getEnbLayerOptions(){
		var sizeFunction = new L.LinearFunction([1, 16], [253, 48]);
		
		var options = {
				type: 'enb',
				recordsField: null,
				locationMode: L.LocationModes.LATLNG,
				latitudeField: 'lat',
				longitudeField: 'lon',
				displayOptions: {
					'direct_flights': {
						color: new L.HSLHueFunction([0, 200], [253, 330], {
							outputLuminosity: '60%'
						})
					},
					'code': {
						title: function (value) {
							return value;
						}
					}
				},
				filter: function (record) {
					return true;
				},
				setIcon: function (record, options) {// 自定义节点图标
					var html = '<div class="node-item-leaf"><span class="node-code">' + record.code + '</span><i class="el-icon el-icon-topo-enb white-bg" style="font-size: 30px;"></i></div>';

					if(record.type == 'cpe') {
						html = '<div class="node-item-leaf"><span class="node-code">' + record.code + '</span><i class="el-icon el-icon-topo-cpe white-bg" style="font-size: 30px;"></i></div>';
					}
					
					var $html = $(html);

					$html.prepend('<span class="cell-code" style="display: none;">' + record.cellCode + '</span>');
					// 告警信息
					if(record.type == 'enb') {
						var alarmObj = record.alarm;
						if(record.alarm) {
							var keys = ['31001','31002','31003','31004'],
								keycls = [],
								alarmCount = 0,
								levels = {
									31001: 'Critical',
									31002: 'Major',
									31003: 'Minor',
									31004: 'Warning',
								};

							keys.map(function(key){
								if(alarmObj[key]) {
									keycls.push(levels[key]);
									alarmCount += alarmObj[key];
								}
							});

							if(alarmCount) $html.prepend('<span class="alarm-info '+ keycls.join(' ') +'">'+ alarmCount +'</span>');
						}
					}
					// 生成重复GPS节点信息
					var latlonKey = record.lat+'-'+record.lon,
						repeatedNodes = topovm.repeatedMap[latlonKey]||'',
						listText = '';
					if(repeatedNodes.length) {
						repeatedNodes.map(function(item){
							listText += '<span class="repeated-item">' + item + '</span>';
						});
						listText = '<div class="repeated-list" onclick="event.stopPropagation()">' + listText + '</div>';

						listText += '<span class="repeated-count" onclick="showRepeatedList(this,event)">+'+ repeatedNodes.length +'</span>';

						$html.prepend('<span class="repeated-nodes">'+ listText +'</span>');
					}
					
					// 连接状态信息
					if(record.online == 'off'){
						//$html.prepend('<span class="connect-status">×</span>');
						$html.addClass('offline');
					}
					//激活状态
					var enbI = $html.find('.icon-enb');
					if(record.active == 'no' && enbI.length) {
						enbI.css({opacity: 0.6});
					}
					
					var $i = $html.find('i');
	
					L.StyleConverter.applySVGStyle($i.get(0), options);
	
					var directFlights = L.Util.getFieldValue(record, 'direct_flights');
					var size = sizeFunction.evaluate(directFlights);
	
					var $code = $html.find('.code');
	
					$code.width(size);
					$code.height(size);
					$code.css('line-height', size + 'px');
					$code.css('font-size', size / 3 + 'px');
					$code.css('margin-top', -size / 2 + 'px');
	
					var icon = new L.DivIcon({
						iconSize: new L.Point(size, size),
						iconAnchor: new L.Point(size / 2, size * 1.5),
						className: 'airport-icon',
						html: $html.wrap('<div/>').parent().html()
					});
	
					return icon;
				},
				onEachRecord: function (layer, record) {
					layer.off('click').on('click', function (evt) {
						// 节点高亮
						highlightNode(record.code);
						if($(evt.originalEvent.target).hasClass('alarm-info')){
							 /* $.extend(topoRecord,{
							 	small_cell_code: record.cellCode
							 }); */
							
							 setTimeout(function(){
								 $('#map').click();
							 },0);
							 setTimeout(function(){
							 	jumpToAliveAlarm(record.alarmLevel, record.code);
							 },15);
						}

						if($(evt.originalEvent.target).hasClass('el-icon-topo-enb')){
							gotoEnbDetailPage(record);
						}
						// if($(evt.originalEvent.target).hasClass('el-icon-topo-cpe')){
						// 	gotoCpeDetailPage(record);
						// }
					});
					$(window).resize();
				}
			};
        return options;
    }

	function gotoEnbDetailPage(record) {
		var cellCode = record.cellCode,
			status = record.online;

		eventAllBus.$emit("gomenupage","1001","",'1001',{},function(){
			setTimeout(function(){
				goCellDetailParamInfoWin("enbStatistics", cellCode, status,'jump');
			},0);
		});
	}
	
	function gotoCpeDetailPage(record) {

	}
	/**
	* 跳转到告警菜单
	* @param alarm_severity{string}：告警级别Id
	* @param sn{string}：基站SN
	**/
	function jumpToAliveAlarm(alarm_severity,sn) {
    	var title = "";
    	if (alarm_severity == '31001') {
    		title = "Critical Alarms";
    	} else if (alarm_severity == '31002') {
    		title = "Major Alarms";
    	} else if (alarm_severity == '31003') {
    		title = "Minor Alarms";
    	} else if (alarm_severity == '31004') {
    		title = "Warning Alarms";
    	}
    	
    	try{
    		eventAllBus.$emit("gomenupage","20044","","20044",{search_text: sn});
    	}catch(e){}
    }
	/**
	* 格式化节点详情信息
	* @param node{object}：节点
	**/
	function showNodeInfo(node){
		var nodeInfo = $.extend({},node);
		var str = [
				'<div class="panelDefault" style="width: 300px;height: 200px;">',
				'<div class="tabsTitle">',
					'<span tabtit="infoTab" onclick="turnTabs(this)" class="active"><%=rb.getString("XinXi")%></span>',
					'<span tabtit="gpsTab" onclick="turnTabs(this);"><%=rb.getString("WeiZhi")%></span>',
				'</div>',
				'<div class="el-icon el-icon-close node-info-close" onclick="cancelSet()"></div>',
				'<div class="tabsContentDiv">',
					'<div class="infoTab" style="display: block;">',
						'<ul class="node-ul">',
						'<li><span class="node-label"><%=rb.getString("DianYuanBianMa")%><%=rb.getString("MaoHao")%></span> {code}</li>',
						'<li><span class="node-label"><%=rb.getString("HostName")%><%=rb.getString("MaoHao")%></span> {name}</li>',
						'<li><span class="node-label">IP<%=rb.getString("MaoHao")%></span> {ip}</li>',
						'<li><span class="node-label"><%=rb.getString("SheBeiZu")%><%=rb.getString("MaoHao")%></span> {groupName}</li>',
						'<li><span class="node-label"><%=rb.getString("JiHuoZhuangTai")%><%=rb.getString("MaoHao")%></span> {active}</li>',
						'<li><span class="node-label"><%=rb.getString("WeiZhi")%><%=rb.getString("MaoHao")%></span> {loc}</li>',
						'<li style="display: flex"><span class="node-label"><%=rb.getString("MMEZhuangTai")%><%=rb.getString("MaoHao")%></span> {mmeFormat}</li>',
						'<ul>',
					'</div>',
					'<div class="gpsTab" style="padding: 10px 20px;">',
						'<ul class="node-ul">',
						'<li><%=rb.getString("JingDu")%><%=rb.getString("MaoHao")%></li>',
						'<li><input type="number" class="border-box border" value="{lon}" placeholder="<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> -180 -- 180"/></li>',
						'<li> </li>',
						'<li><%=rb.getString("WeiDu")%><%=rb.getString("MaoHao")%></li>',
						'<li><input type="number" class="border-box border" value="{lat}" placeholder="<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> -90 -- 90"/></li>',
						'<li> </li>',
						'<ul>',
						'<div style="padding-top: 10px;" class="gps-setting-op">',
							'<button class="el-button el-button--primary" onclick="setNodeGPS(&quot;'+node.cellCode+'&quot;,this)"><%=rb.getString("QueDing")%></button>',
							'<button class="el-button" onclick="cancelSet()"><%=rb.getString("QuXiao")%></button>',
						'</div>',
					'</div>',
				'</div>',
				'</div>'
			];

		if(node.type=='cpe') {
			str = [
				'<div class="panelDefault" style="width: 300px;height: 200px;">',
				'<div class="tabsTitle">',
					'<span tabtit="infoTab" onclick="turnTabs(this)" class="active"><%=rb.getString("XinXi")%></span>',
					'<span tabtit="gpsTab" onclick="turnTabs(this);"><%=rb.getString("WeiZhi")%></span>',
				'</div>',
				'<div class="el-icon el-icon-close node-info-close" onclick="cancelSet()"></div>',
				'<div class="tabsContentDiv">',
					'<div class="infoTab" style="display: block;">',
						'<ul class="node-ul">',
						'<li><span class="node-label"><%=rb.getString("CPEXuLieHao")%><%=rb.getString("MaoHao")%></span> {sn}</li>',
						'<li><span class="node-label"><%=rb.getString("CPEName")%><%=rb.getString("MaoHao")%></span> {name}</li>',
						'<li><span class="node-label"><%=rb.getString("IMSI")%><%=rb.getString("MaoHao")%></span> {imsi}</li>',
						'<li><span class="node-label">IP<%=rb.getString("MaoHao")%></span> {ip}</li>',
						'<li><span class="node-label"><%=rb.getString("MACDiZhi")%><%=rb.getString("MaoHao")%></span> {mac}</li>',
						'<li><span class="node-label"><%=rb.getString("SheBeiZu")%><%=rb.getString("MaoHao")%></span> {groupName}</li>',
						'<li><span class="node-label"><%=rb.getString("WeiZhi")%><%=rb.getString("MaoHao")%></span> {loc}</li>',
						'<ul>',
					'</div>',
					'<div class="gpsTab" style="padding: 10px 20px;">',
						'<ul class="node-ul">',
						'<li><%=rb.getString("JingDu")%><%=rb.getString("MaoHao")%></li>',
						'<li><input type="number" class="border-box border" value="{lon}" placeholder="<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> -180 -- 180"/></li>',
						'<li> </li>',
						'<li><%=rb.getString("WeiDu")%><%=rb.getString("MaoHao")%></li>',
						'<li><input type="number" class="border-box border" value="{lat}" placeholder="<%=rb.getString("FanWei")%><%=rb.getString("MaoHao")%> -90 -- 90"/></li>',
						'<li> </li>',
						'<ul>',
						'<div style="padding-top: 10px;" class="gps-setting-op">',
							'<button class="el-button el-button--primary" onclick="setNodeGPS(&quot;'+node.code+'&quot;,this)"><%=rb.getString("QueDing")%></button>',
							'<button class="el-button" onclick="cancelSet()"><%=rb.getString("QuXiao")%></button>',
						'</div>',
					'</div>',
				'</div>',
				'</div>'
			];
		}
		
		nodeInfo.active = node.active == 'yes'? '<%=rb.getString("JiHuo")%>':'<%=rb.getString("QuJiHuo")%>';
		if(nodeInfo.lat || nodeInfo.lon) {
			nodeInfo.loc = nodeInfo.olon + ',' + nodeInfo.olat;
		}else {
			nodeInfo.loc = '';
		}
		// 格式化mme
		if(node.type=='enb') {
			var rowData = {
					s1siglinkserverlist: nodeInfo.serverList,
					mmeEnable: nodeInfo.mmeEnable,
					mmeStatus: nodeInfo.mmeStatus,
					mme_pool_1: nodeInfo.mmePool1,
					mme_pool_2: nodeInfo.mmePool2
				};
			nodeInfo.mmeFormat = mmeStatusFormatter(nodeInfo.mmeStatus, rowData);
		}
		
		return str.join(' ').evaluate(nodeInfo);
	}
	/**
	* 获取enb全量图层配置项
	* @param eNodebsLookup{array}：enb节点队列
	**/
	function getAllLayerOptions(eNodebsLookup){
		var maxCount = Number(0);
		
		// 获取节点位置信息
		var getLocation = function (context, locationField, fieldValues, callback) {
			var key = fieldValues[0];
			var enodeb = eNodebsLookup[key];
			var location;

			if (enodeb) {
				var latlng = new L.LatLng(Number(enodeb.lat), Number(enodeb.lon));

				location = {
					location: latlng,
					text: key,
					center: latlng
				};
			}

			return location;
		};
		var options = {
			recordsField: null,
			locationMode: L.LocationModes.CUSTOM,
			fromField: 'enb',
			toField: 'tar',
			codeField: null,
			getLocation: getLocation,
			getEdge: L.Graph.EDGESTYLE.ARC,
			includeLayer: function (record) { // 控制默认是否显示连线
				return true;
			},
			getIndexKey: function (location, record) {
				return record.enb + '_' + record.tar;
			},
			setHighlight: function (style) {
				style.opacity = 1.0;

				return style;
			},
			unsetHighlight: function (style) {
				style.opacity = 0.5;
				
				return style;
			},
			layerOptions: {
				fill: false,
				opacity: 0.5,
				weight: 0.5,
				fillOpacity: 1.0,
				color: '#1DA3FC',
				distanceToHeight: new L.LinearFunction([0, 20], [1000, 300]),
				markers: {
					end: true
				},

				// Use Q for quadratic and C for cubic
				mode: 'Q'
			},
			tooltipOptions: {
				iconSize: new L.Point(80, 64),
				iconAnchor: new L.Point(-5, 64),
				className: 'leaflet-div-icon line-legend hidden'
			},
			displayOptions: {
				alarmCount: {
					weight: new L.LinearFunction([0, 1], [maxCount, 14]),
					color: new L.HSLHueFunction([0, 200], [maxCount, 330], {
						outputLuminosity: '60%'
					}),
					displayName: ' '
				}
			},
			onEachRecord: function (layer, record) {
				(function(lay,rec){
					setTimeout(function(){
						if(!isConnected(rec)) {// 连接状态颜色
							lay.setStyle({
								color: '#E64242'
							})
						}
					},0);
				})(layer,record)
				//layer.bindPopup($(L.HTMLUtils.buildTable(record)).wrap('<div/>').parent().html());
			}
		};
		return options;
	}
	/**
	* 是否连接正常
	* @param record{object}：地图节点数据
	**/
	function isConnected(record){
		var bool = false;
		if(record.label == 'EPC'){// EPC连线
			if(record.mmeEnable) {
				bool = record.mmeStatus == 1;
			}else if(record.mmeStatus){
				var poolArr = record.mmeStatus.split(','); // ['mme1=1','mme2=0']格式
				poolArr.map(function(item){
					if(item){
						var arr = item.split('=');
						if(arr[0] == record.tar && arr[1] == 1) bool = true;
					}
				})
			}
		}else if(record.label == 'OMC'){// OMC连线
			if(record.online == 'on') bool = true;
		}
		
		return bool;
	}
	/**
	* MME状态-内容处理
	* @param value{string}：mme状态值
	* @param rowData{object}：节点数据
	* @param rowIndex{number}：对应下标
	**/
	function mmeStatusFormatter(value, rowData, rowIndex){
		if (value == null || value == "") {
			return null;
		}
		
		if (value == "1") {
			value = "<span class='statusIcon status_conn' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()'>MME</span>"
					+ "<div class='mmeDetails MMEDetail'>MME IP : "+rowData.s1siglinkserverlist + "</br>MME Status : <span class='mmeStauts'></span></div>";
		} else if (value == "0") {
			value = "<span class='statusIcon status_disconn' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()'>MME</span>"
			+ "<div class='mmeDetails MMEDetail'>MME IP : "+rowData.s1siglinkserverlist + "</br>MME Status : <span class='mmeStauts'></span></div>";
		} else if (value == "2") {
			value = "--";
		} else{
			var mmePools = value.split(",");
			var ret = "";
			for(var i = 0; i < mmePools.length; i++ ){
				var mme = mmePools[i];
				var mmeArr = mme.split("=");
				
				if(mmeArr.length == 2 && "mme1" == mmeArr[0] && mmeArr[1] == "1"){
					<%-- ret = ret + "MMEStatus1 "+"<%= rb.getString("MMEYiLianJie")%>,"; --%>
					ret = ret + "<span class='statusIcon status_conn' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' >MME1</span>"; 
				}else if(mmeArr.length == 2 && "mme1" == mmeArr[0] && mmeArr[1] == "0"){
					ret = ret + "<span  class='statusIcon status_disconn' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()'>MME1</span>";
				}else if(mmeArr.length == 2 && "mme2" == mmeArr[0] && mmeArr[1] == "1"){
					ret = ret + "<span  class='statusIcon status_conn' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' style='margin-left:10px;'>MME2</span>";
				}else if(mmeArr.length == 2 && "mme2" == mmeArr[0] && mmeArr[1] == "0"){
					ret = ret + "<span class='statusIcon status_disconn' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' style='margin-left:10px;'>MME2</span>"; 
				}
			}
			var lastChar = ret.charAt(ret.length - 1);
			if("," == lastChar){
				ret = ret.substring(0,ret.length - 1);
			}
			value = ret + "<div class='mmeDetails MME1Detail'>MME1 IP : "+rowData.mme_pool_1 + "</br>MME1 Status : <span class='mme1Stauts'></span></div>"+ "<div class='mmeDetails MME2Detail'>MME2 IP : "+rowData.mme_pool_2 + " <br/>MME2 Status : <span class='mme2Stauts'></span></div>";  
		}
		
		return value;
	}
	/**
	* MME状态-内容处理
	* @param code{string}：节点code
	* @param obj{object}： gps数据对象
	**/
	function setNodeGPS(code, obj) {
		var inputs = $(obj).parents('.gpsTab').find('input'),
				arr = [];
		inputs.each(function(idx,item){
			var itemVal = item.value;
			if(itemVal) arr.push(itemVal);
			else arr.push('');
		});

		if(!validPGSValue(arr.join(','))){
			toast('<%=rb.getString("GPSBuHeFa")%>',$('#mainpage'));
			return;
		}
		// 保存节点gps数据
		$.ajax({
			url: '${ctx}/cell/topo/setLocationInfo.action',
			type: 'post',
			dataType: 'json',
			data: {
				longitude: arr[0],
				latitude: arr[1],
				cell_code: code,
				operator_code: operator_code
			},
			success: function(data){
				if(data['success']){
					refreshTopo({
						cellCode: code,
						lat: arr[1],
						lon: arr[0],
						olat: arr[1],
						olon: arr[0]
					});
					topovm.$refs.enbDevice.refresh();
					topovm.$refs.cpeDevice.refresh();
				}else{
					toast(data['message'],$('#omc_app_ctn'));
				}
			}
		})
	}
	// 关闭节点信息浮层
	function cancelSet() {
		$('.leaflet-popup-close-button')[0].click();
	}
	/**
	* 校验经纬度有效性
	* @param gpsValue{object}： gps数据对象
	**/
	function validPGSValue(gpsValue){
		var bool = false;
		if(gpsValue){
			var arr = gpsValue.split(','),
				lon = arr[0]-0,
				lat = arr[1]-0;
			if(!isNaN(lat) && !isNaN(lon)){
				if(Math.abs(lat) <= 90 && Math.abs(lon) <= 180 && lessThenLength(lat,8) && lessThenLength(lon,9) ) {
					bool = true;
				}
			}
		}
		return bool;
	}
	/**
	* 判断字符是否小于指定长度
	* @param str{string}：要判断的字符
	* @param num{number}：长度值
	**/
	function lessThenLength(str,num){
		str += '';
		return str.replace(/\./g,'').length <= num;
	}
	/* 鼠标移出隐藏mme信息 */
	function hideMMEDetail(){
		$(".mmeDetails").fadeOut(300);
	}
	/**
	* 鼠标移入，显示当前MME详细信息 
	* @param ele{dom}：容器dom节点
	* @param index{number}：mme状态值
	**/
	function toMMEDetail(ele,index){
		var thisTop = $(ele).offset().top;
		var allHeight = $(document).height();
		var thisLeft = $(ele).offset().left;
		var allWidth = $(document).width();
		
		if((allHeight - thisTop) < 200){
			$(ele).siblings(".mmeDetails").css("top","-55px");
		}
		if((allWidth - thisLeft) < 200){
			$(ele).siblings(".mmeDetails").css("left","-90px");
		}
		
		var YiLianJie = '<%= rb.getString("MMEYiLianJie")%>';
		var WeiLianJie = '<%= rb.getString("MMEWeiLianJie")%>';
		var eleClass = $(ele).text(); 
		$(".mmeDetails").hide();
		
		if(eleClass == 'MME1'){
			$(ele).siblings(".MME1Detail").fadeToggle();
			if(index == 1){
				$(".mme1Stauts").text(YiLianJie);
			}else if(index == 0){
				$(".mme1Stauts").text(WeiLianJie);
			}
		}else if(eleClass == 'MME2'){
			$(ele).siblings(".MME2Detail").fadeToggle();
			if(index == 1){
				$(".mme2Stauts").text(YiLianJie);
			}else if(index == 0){
				$(".mme2Stauts").text(WeiLianJie);
			}
		}else if(eleClass == 'MME'){
			$(ele).siblings(".MMEDetail").fadeToggle();
			if(index == 1){
				$(".mmeStauts").text(YiLianJie);
			}else if(index == 0){
				$(".mmeStauts").text(WeiLianJie);
			}
		}	
	}
</script>

