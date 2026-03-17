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
		padding: 10px 30px 10px 15px;
		position: absolute; 
		right: 15px; 
		top: 50px;
		background-color: #fff;
		z-index: 10;
		box-shadow: 2px 3px 8px #cdc6c6;
	}
	.topo-legend-cls > div {
		padding: 3px;
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
	.node-info-close {
		position: absolute;
		top: 5px;
		right: 10px;
	}
	.group-arrow {
		padding: 5px 0px;
		position: absolute;
		left: 0;
		top: 50%;
		border-radius: 10px;
		color: #fff;
		background-color: #689CF3;;
		z-index: 10;
	}
	.group-arrow.el-icon-right::before, .group-arrow.el-icon-left::before {
		color: #fff;
	}
	.sas-switch-on .gps-setting-op {
		display: none;
	}
	.sas-switch-on .gpsTab::before {
		content: '';
		position: absolute;
		top: 0;
		right: 0;
		bottom: 0;
		left: 0;
		z-index: 10;
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
		display: block;
		white-space: nowrap;
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
	.top-z-index {
		z-index: 99999 !important;
	}
</style>

<div id="topo_container" style="height: 100%;display: flex;border: 1px solid #E9E9E9;padding: 2px 0px 0px 5px;background: #fff;">
	<!-- Device group -->
	<div style="width: 240px;height: 100%;position: relative;" class="flex-ctn" v-show="groupShow">
		<div class="flex-ctn absolute-ctn">
			<div style="font-weight: bold;padding: 5px 0;font-size: 14px;"><%=rb.getString("SheBeiZu")%></div>
			<div style="flex: auto;" @click="observSlider">
				<el-ctable ref="group" :pagination="false" :rownumber="false" :query-params="groupQueryParams" :url="groupURL" @selection-change="groupChange" row-key="id"
					@load-success="loadSuccess">
					<template slot="toolbar">
						<el-query class="width-mini" type="normal" @query="queryGroup" placeholder="<%=rb.getString("SheBeiZuMingCheng")%>"></el-query>
					</template>
					<el-table-column type="selection"></el-table-column>
					<el-table-column label="<%=rb.getString("SheBeiZuMingCheng")%>" prop="group_name">
						<template slot-scope="scope">
							<div style="display: flex;align-items: center;justify-content:space-between;">
								<span class="ellipsis-txt" style="width: 150px;">{{scope.row.group_name}}</span>
								<i class="row-op el-icon el-icon-arrow-right" @click="rowClick(scope.row)"></i>
							</div>
						</template>
					</el-table-column>
				</el-ctable>
			</div>
		</div>
		<!-- device slider -->
		<div ref="deviceSlider" class="device-slider flex-ctn">
			<div>
				<el-tabs v-model="activeName">
					<el-tab-pane label="<%=rb.getString("XiaoZhan")%>" name="ENB"></el-tab-pane>
					<el-tab-pane label="<%=rb.getString("CPE")%>" name="CPE" class="CODE_CPE_MONITOR hidden"></el-tab-pane>
				</el-tabs>
			</div>
			<!-- device list -->
			<div style="flex: auto;overflow: auto;">
				<el-ctable ref="enbDevice" v-show="activeName=='ENB'" @row-dblclick="rowdblclick" :rownumber="false" :url="deviceURL" :query-params="enbQueryParams" :front-pagination="true">
					<template slot="toolbar">
						<el-query ref="enblist" class="width-mini" type="normal" @query="queryDevice" placeholder="<%=rb.getString("SheBeiMingCheng")%>/<%=rb.getString("HostName")%>"></el-query>
					</template>
					<el-table-column width="40">
						<template slot-scope="scope">
							<div v-if="!sasEnable" class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose"></div>
							<div v-if="sasEnable" class="el-icon el-icon-operation-more disabled"></div>
						</template>
					</el-table-column>
					<el-table-column label="<%=rb.getString("DianYuanBianMa")%>" prop="serial_number" min-width="120"></el-table-column>
					<el-table-column label="<%=rb.getString("HostName")%>" prop="host_name"></el-table-column>
					<el-table-column label="<%=rb.getString("JingDu")%>" prop="longitude"></el-table-column>
					<el-table-column label="<%=rb.getString("WeiDu")%>" prop="latitude"></el-table-column>
				</el-ctable>

				<el-ctable ref="cpeDevice" v-show="activeName=='CPE'" @row-dblclick="rowdblclick" :rownumber="false" :url="deviceURL" :query-params="cpeQueryParams" :front-pagination="true">
					<template slot="toolbar">
						<el-query ref="cpelist" class="width-mini" type="normal" @query="queryDevice" placeholder="<%=rb.getString("SheBeiMingCheng")%>/<%=rb.getString("CPEName")%>/IMSI"></el-query>
					</template>
					<el-table-column width="40">
						<template slot-scope="scope">
							<div v-if="!sasEnable" class="el-icon el-icon-operation-more" @click="optClick(scope.row,event)" v-clickoutside="handerClose"></div>
							<div v-if="sasEnable" class="el-icon el-icon-operation-more disabled"></div>
						</template>
					</el-table-column>
					<el-table-column label="<%=rb.getString("DianYuanBianMa")%>" prop="serial_number" min-width="150"></el-table-column>
					<el-table-column label="<%=rb.getString("MACDiZhi")%>" prop="macaddress" min-width="130"></el-table-column>
					<el-table-column label="<%=rb.getString("CPEName")%>" prop="host_name" min-width="100"></el-table-column>
					<el-table-column label="<%=rb.getString("IMSI")%>" prop="imsi" min-width="140"></el-table-column>
					<el-table-column label="<%=rb.getString("JingDu")%>" prop="longitude" min-width="90"></el-table-column>
					<el-table-column label="<%=rb.getString("WeiDu")%>" prop="latitude" min-width="90"></el-table-column>
				</el-ctable>

				<el-cmenu ref="menu" :data="menus" @click="eventClick"></el-cmenu>
			</div>
		</div>
	</div>
	<!-- topo map -->
	<div style="flex: auto; display: flex; flex-direction: column;position: relative;background: #fff;">
		<div :class="arrowClass" @click="groupShow = !groupShow"></div>
		<!-- 操作按钮 -->
		<div style="min-height: 30px;">
			<div style="padding: 8px;overflow: hidden;text-overflow:ellipsis;white-space:nowrap;width: 300px;color: #363B4E;font-size: 14px;">{{groupNames}}</div>
			<el-popover>
				<div class="circleIcon placeholder-bt" slot="reference" tip="<%=rb.getString("SheZhi")%>" style="display: inline-block;top: 0;right: 50px;">
					<i class="el-icon el-icon-circle-display"></i>
				</div>
				<el-form style="width: 500px;padding: 15px;" label-width="80" class="form-item-bottom-15">
					<el-form-item label="<%=rb.getString("KPISheBei")%>:">
						<el-checkbox-group v-model="statusForm.device">
							<el-checkbox label="enb"><%=rb.getString("XiaoZhan")%></el-checkbox>
							<el-checkbox label="cpe" v-show="isCPEUsable"><%=rb.getString("CPE")%></el-checkbox>
						</el-checkbox-group>
					</el-form-item>

					<el-form-item label="<%=rb.getString("LianXianXianShi")%>:" v-show="isCPEUsable">
						<el-checkbox-group v-model="statusForm.line">
							<el-checkbox label="yes"><%=rb.getString("XiaoZhan")%>-<%=rb.getString("CPE")%></el-checkbox>
						</el-checkbox-group>
					</el-form-item>

					<el-form-item label="<%=rb.getString("ZhuangTai")%>:">
						<div class="flex-align-center">
							<span><%=rb.getString("XiaoZhan")%>:&nbsp;&nbsp;</span>
							<el-checkbox-group v-model="statusForm.enbStatus">
								<el-checkbox label="on"><%=rb.getString("ZaiXian")%></el-checkbox>
								<el-checkbox label="off"><%=rb.getString("LiXian")%></el-checkbox>
								<el-checkbox label="yes"><%=rb.getString("JiHuo")%></el-checkbox>
								<el-checkbox label="no"><%=rb.getString("QuJiHuo")%></el-checkbox>
							</el-checkbox-group>
						</div>
						<div v-show="isCPEUsable" class="flex-align-center">
							<span><%=rb.getString("CPE")%>:&nbsp;&nbsp;</span>
							<el-checkbox-group v-model="statusForm.cpeStatus">
								<el-checkbox label="on"><%=rb.getString("ZaiXian")%></el-checkbox>
								<el-checkbox label="off"><%=rb.getString("LiXian")%></el-checkbox>
							</el-checkbox-group>
						</div>
					</el-form-item>

					<el-form-item label="<%=rb.getString("GaoJing")%>:">
						<div class="flex-align-center">
							<span><%=rb.getString("XiaoZhan")%>:&nbsp;&nbsp;</span>
							<el-checkbox-group v-model="statusForm.enbAlarm">
								<el-checkbox label="31001">Critical</el-checkbox>
								<el-checkbox label="31002">Major</el-checkbox>
								<el-checkbox label="31003">Minor</el-checkbox>
								<el-checkbox label="31004">Warning</el-checkbox>
							</el-checkbox-group>
						</div>
					</el-form-item>

					<div>
						<el-button type="primary" size="mini" @click="setStatus"><%=rb.getString("QueDing")%></el-button>
						<el-button size="mini" @click="cancelSet"><%=rb.getString("QuXiao")%></el-button>
					</div>
				</el-form>
			</el-popover>
			<div class="circleIcon placeholder-bt" tip="<%=rb.getString("QuanPin")%>" style="display: inline-block;top: 0;">
				<i class="el-icon el-icon-circle-open" @click="FullScreen"></i>
			</div>
		</div>
		<!-- 图例 -->
		<div class="topo-legend-cls">
			<div style="font-weight: bold;"><%=rb.getString("SheBeiZhuangTai")%></div>
			<div><i class="el-icon el-icon-topo-enb"></i> <%=rb.getString("XiaoZhan")%> - <%=rb.getString("ZaiXian")%></div>
			<div v-show="isCPEUsable"><i class="el-icon el-icon-topo-cpe online"></i> <%=rb.getString("CPE")%> - <%=rb.getString("ZaiXian")%></div>
			<div><i class="el-icon el-icon-topo-enb offline"></i> <%=rb.getString("XiaoZhan")%> - <%=rb.getString("LiXian")%></div>
			<div v-show="isCPEUsable"><i class="el-icon el-icon-topo-cpe offline"></i> <%=rb.getString("CPE")%> - <%=rb.getString("LiXian")%></div>
		</div>
		<!-- topo 地图 -->
		<div style="flex: auto;overflow: auto;" :class="sasSwitchCls">
			<div id="map" style="width: 100%;min-height: 100%;background: #A1D4E0;"></div>
		</div>
		<!-- 状态数值 -->
		<div id="topo_cell_statis" style="height: 30px; line-height: 30px;text-align: right;padding-right: 10px;">
			<span><%=rb.getString("LianJieZhuangTai")%><%=rb.getString("MaoHao")%></span>
        	<span class="connStatusStatistics-gps"></span>
        
        	<span style="margin-left: 50px;"><%=rb.getString("ShiFouJiHuo")%><%=rb.getString("MaoHao")%></span>
        	<span class="opStatusStatistics-gps"></span>
        
           <span style="margin-left: 50px;"><%=rb.getString("MMEZhuangTai")%><%=rb.getString("MaoHao")%></span>
           <span class="mmeStatusStatistics-gps"></span>
		</div>
	</div>

	<el-dialog ref="gpsdg" title="<%=rb.getString("WeiZhi")%>" :visible.sync="gpsdgVisible" :width="400">
		<el-form ref="gpsform" :model="gpsForm" :rules="gpsRules" label-position="top">
			<el-form-item v-if="activeName == 'ENB'">
				SN<%=rb.getString("MaoHao")%> {{curRow.serial_number}}
			</el-form-item>
			<el-form-item v-else>
				<%=rb.getString("MACDiZhi")%><%=rb.getString("MaoHao")%> {{curRow.macaddress}}
			</el-form-item>
			
			<el-form-item label="<%=rb.getString("JingDu")%><%=rb.getString("MaoHao")%>" prop="lon">
				<el-input v-model="gpsForm.lon"></el-input>
			</el-form-item>
			<el-form-item label="<%=rb.getString("WeiDu")%><%=rb.getString("MaoHao")%>" prop="lat">
				<el-input v-model="gpsForm.lat"></el-input>
			</el-form-item>
		</el-form>
		<div>
			<el-button type="primary" @click="setGPS"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="gpsdgVisible = false"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
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
			var vm = this,
				/**
				* 纬度校验
				* @param rule{object}：校验配置的规则
				* @param value{string}: 纬度值
				* @param cb{function}：回调方法
				**/
				latValid = function(rule,value,cb){
					if(value) {
						if(isNaN(value)) {
							cb('<%=rb.getString("WeiDuFanWei")%>: [-90,90], <%=rb.getString("JingQueDu")%>: 6');
						}else if(value<-90 || value>90){
							cb('<%=rb.getString("WeiDuFanWei")%>: [-90,90], <%=rb.getString("JingQueDu")%>: 6');
						}else {
							var arr = value.split('.'),
								precision = arr[1]||'';
							if(precision.length>6) {
								cb('<%=rb.getString("WeiDuFanWei")%>: [-90,90], <%=rb.getString("JingQueDu")%>: 6');
							}else cb();
						}
					}else {
						cb();
					}
				},
				/**
				* 经度校验
				* @param rule{object}：校验配置的规则
				* @param value{string}: 经度值
				* @param cb{function}：回调方法
				**/
				lonValid = function(rule,value,cb){
					if(value) {
						if(isNaN(value)) {
							cb('<%=rb.getString("JingDuFanWei")%>: [-180,180], <%=rb.getString("JingQueDu")%>: 6');
						}else if(value<-180 || value>180){
							cb('<%=rb.getString("JingDuFanWei")%>: [-180,180], <%=rb.getString("JingQueDu")%>: 6');
						}else {
							var arr = value.split('.'),
								precision = arr[1]||'';
							if(precision.length>6) {
								cb('<%=rb.getString("JingDuFanWei")%>: [-180,180], <%=rb.getString("JingQueDu")%>: 6');
							}else cb();
						}
					}else {
						cb();
					}
				};
			
			return {
				limit: 100, // 过滤数据: 阀值
				links: {}, // enb为key，cpe存于数组中：enbCode:[cpeCode1, cpeCode2, ...]，建立 1:n 关系
				activeName: 'ENB',
				groupURL: '${ctx}/system/deviceGroup/getDeviceGroupList.action',
				deviceURL: '',
				// 设备组查询参数
				groupQueryParams: {
					search_text: ''
				},
				// enb列表查询参数
				enbQueryParams: {
					device_group_id: '',
					device_type: 'ENB',
					search_text: '',
					operator_code: operator_code
				},
				// cpe列表查询参数
				cpeQueryParams: {
					device_group_id: '',
					device_type: 'CPE',
					search_text: '',
					operator_code: operator_code
				},
				// 显示状态表单 -- 提交用
				form: {
					device: ['enb','cpe'],
					line: ['yes'],
					enbStatus: ['on','off','yes','no'],
					cpeStatus: ['on','off'],
					enbAlarm: ['31001', '31002', '31003', '31004']
				},
				// 显示状态表单 -- 界面展示用
				statusForm: {
					device: ['enb','cpe'],
					line: ['yes'],
					enbStatus: ['on','off','yes','no'],
					cpeStatus: ['on','off'],
					enbAlarm: ['31001', '31002', '31003', '31004']
				},
				// gps设置表单
				gpsForm: {
					lat: '', // 纬度
					lon: ''
				},
				// gps域校验规则
				gpsRules: {
					lat:[
						{validator: latValid}
					],
					lon:[
						{validator: lonValid}
					]
				},
				cpeNodes: [], // 记录获取的cpe节点
				enbNodes: [], // 记录获取的enb节点
				prevNodes: [], // 缓存的上次绘制节点
				menus: [],
				gpsdgVisible: false,
				allNodes: [], // cpe、enb节点合集
				curDeviceCode: '',
				curSerialNumber: '',
				sasEnable: true,
				groupShow: true,
				curRow: {},
				groupNames: '',
				firtLoad: true,
				repeatedMap: {},
				repeatedHideNodes: []
			};
		},
		computed: {
			// 设备组展开设备列表时 箭头样式
			arrowClass() {
				var vm = this;

				return {
					'group-arrow': true,
					'el-icon': true,
					'el-icon-left':  vm.groupShow,
					'el-icon-right':  !vm.groupShow
				}
			},
			// sas开关状态样式 -- 控制是否可修改gps
			sasSwitchCls() {
				var vm = this;

				return  {
					'sas-switch-on': vm.sasEnable
				}
			},
			isCPEUsable() {
				return writableMap['CODE_CPE_MONITOR'] === true;
			}
		},
		watch: {
			form: {
				deep: true,
				handler(n) {
					var vm = this;
					
					vm.filterRender(vm.allNodes, globMap, true)
				}
			},
			activeName: function(val){
				var vm = this;

			}
		},
		methods: {
			/**
			* 双击列表行 自动定位topo中节点（如果在tops中）
			* @param row{object}: 列表行数据
			* @param evt{event}：鼠标事件
			**/
			rowdblclick(row,evt) {
				var code = row.small_cell_code?row.serial_number:row.cpe_code,
					valid = this.allNodes.map(function(row){
						return row.code;
					}).includes(code);
				// topo中有该节点时，高亮居中显示
				if(valid && row.latitude && row.longitude){
					globMap.setView([row.latitude-0, row.longitude-0], 3);
					setTimeout(function(){
						highlightNode(code)
					},500)
				}
			},
			/**
			* 设备组列表数据加载成功回调
			* @param req{object}: 列表加载返回数据
			**/
			loadSuccess(req) {
				var vm = this,
					rows = req.rows,
					row = '';
				
				if(rows && rows.length) row = rows[0];
				// 首次加载默认选中第一行
				if(vm.firtLoad) {
					setTimeout(function(){
						vm.$refs.group.toggleRowSelection(row,true);
						vm.$refs.group.select([row],row);
						vm.firtLoad = false
					},10);
				}
			},
			/**
			* 设备组列表查询
			* @param text{string}: 检索文本
			**/
			queryGroup(text) {
				var vm = this;

				vm.groupQueryParams.search_text = (text||'').trim();
			},
			/**
			* 设备列表查询
			* @param text{string}: 检索文本
			**/
			queryDevice(text) {
				var vm = this,
					stxt = (text||'').trim();

				if(vm.activeName == 'ENB') {
					vm.enbQueryParams.search_text = stxt;
				}else {
					vm.cpeQueryParams.search_text = stxt;
				}
			},
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
			/**
			* 设备组选择项变动回调
			* @param selection{array}: 当前选择项
			**/
			groupChange(selection) {
				var vm = this;
				
				vm.groupNames = (selection||[]).map(function(item){ return item.group_name;}).join(', ');
				var ids = (selection||[]).map(function(item){ return item.id;});
				ids = ids.join(',');
				// 刷新topo和统计信息数据
				vm.getTopoInfos(ids);
				vm.refresh_topo_Statistics(ids);
			},
			// 状态控制设置
			setStatus() {
				var vm = this;

				Object.assign(vm.form, vm.statusForm);
				vm.cancelSet();
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
							vm.repeatedHideNodes.push(node);
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
			getTopoInfos(ids) {
				var vm = this,
					url = '${ctx}/cell/topo/getDeviceInfoList.action',
					params = {
						queryType: 'groupid',
						device_group_id: ids,
						device_type: 'ENB',
						operator_code: operator_code
					};
				vm.links = {};
				// 获取enb节点
				
				vm.repeatedMap = {};
				vm.repeatedHideNodes = [];
				
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
							vm.initMap();
						});
					}else {
						vm.initMap();
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
			
				return cpes;
			},
			/**
			* 设备组展开箭头点击事件
			* @param row{object}: 设备组列表行数据
			**/
			rowClick(row) {
				var vm = this;
				// 刷新设备列表数据
				vm.enbQueryParams.device_group_id = row.id;
				vm.cpeQueryParams.device_group_id = row.id;
				vm.$nextTick(function(){
					vm.deviceURL = '${ctx}/cell/topo/getDeviceInfoList.action';
				});
			},
			/**
			* 设备列表更多操作点击事件
			* @param row{object}: 设备列表行数据
			* @param ev{event}: 鼠标事件
			**/
			optClick(row,ev){
    	    	var vm = this,
					status = row.status,
    	    		isStopShow = status == 'on',
					cellCode = vm.activeName == 'ENB'?row.small_cell_code:row.cpe_code,
					serialNumber = row.serial_number;
			
				vm.curRow = row;
		    	this.menus= [
			          {label:'<%=rb.getString("WeiZhi")%>', code:'view', cls:"el-icon el-icon-operation-location",serialNumber: serialNumber,sn: cellCode,lat: row.latitude,lon: row.longitude},
			    ];
				// 展示菜单
		    	var vm = this;
		    	this.$nextTick(function(){
		    		document.body.click();
    		    	vm.$refs.menu.show(ev);
		    	});
    	    },
			/**
			* 菜单点击事件
			* @param row{object}: 设备列表行数据
			**/
			eventClick(row) {
				var vm = this;
				
				vm.gpsdgVisible = true;
				vm.curDeviceCode = row.sn;
				vm.curSerialNumber = row.serialNumber
				vm.$nextTick(function(){
					vm.$refs.gpsform.resetFields();
					Object.assign(vm.gpsForm,{lat:row.lat,lon: row.lon});
				})
			},
			// 设置经纬度
			setGPS() {
				var vm = this,
					url = '${ctx}/cell/topo/setLocationInfo.action',
					params = {
						cell_code: vm.curDeviceCode,
						longitude: vm.gpsForm.lon,
						latitude: vm.gpsForm.lat,
						operator_code: operator_code
					};
				// 保存设备经纬度信息
				vm.$refs.gpsform.validate(function(r){
					if(r) {
						axios.post(url,stringify(params)).then(function(res){
							if(res.data['success']){
								vm.$refs.enbDevice.refresh();
								vm.$refs.cpeDevice.refresh();
								vm.gpsdgVisible = false;

								var codes = vm.allNodes.map(function(item){return item.cellCode;});
								if(codes.includes(vm.curDeviceCode)) {
									refreshTopo({
										code: vm.curSerialNumber,
										cellCode: vm.curDeviceCode,
										lat: params.latitude,
										lon: params.longitude,
										olat: params.latitude,
										olon: params.longitude
									});
								}else {
									var ids = vm.$refs.group.getChecked().join(',');
									vm.getTopoInfos(ids);
								}
							}else{
								toast(res.data['message'],$('#omc_app_ctn'));
							}
						})
					}
				})
			},
			// 点击页面其他地方菜单收起
			handerClose(){
    	        this.$refs.menu.hide();
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
				if(!vm.form.device.includes('cpe')) return;

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
			* 根据设置面板过滤节点
			* @param nodes{array}: 所有节点
			**/
			filterNodesByStatus(nodes) {
				var vm = this,
					device = vm.form.device||[],
					status = {
						enb: vm.form.enbStatus||[],
						cpe: vm.form.cpeStatus||[]
					},
					levels = ['31001','31002','31003','31004'];
				// 设备类型和状态过滤
				nodes = nodes.filter(function(node){
					var type = node.type,
						online = node.online,
						active = node.active;
					if(node.type=='enb') {
						return device.includes(type) && status[type].includes(online) && status[type].includes(active);
					}else {
						return device.includes(type) && status[type].includes(online);
					}
				});
				// 告警数据过滤
				nodes = nodes.map(function(item){
					var node = Object.assign({},JSON.parse(JSON.stringify(item)));
					if(node.alarm) {
						levels.map(function(key){
							if(!vm.form.enbAlarm.includes(key)) {
								node.alarm[key] = null;
							}
						});
					}

					return node;
				})
				
				return nodes;
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
				if(vm.form.line.length) vm.createLineLayer(nodes,eNodebsLookup,map);
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
				nodes = vm.filterNodesByStatus(nodes);
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
				$(window).on('resize',resize);
				resize();
				
				if(globMap) globMap.remove();
				
				var url = 'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png';
				
				// 处理基站数据格式
				var resObj = vm.proccessNodes(eNodebs);
				// 获取要展示的区域，fitBounds会自动调整合理显示
				var bounds = vm.getViewBounds(resObj);

				if(offlineMapEnable) {
					url = offlineMapServer + '/{z}/{x}/{y}.png';
					// 设置中心位置和放大倍数
					map = L.map('map',{maxZoom: 12,minZoom: 2}).fitBounds(bounds); // .setMaxBounds([[-360,-360],[360,360]])
				}else {
					// 设置中心位置和放大倍数
					map = L.map('map',{maxZoom: 20,minZoom: 2}).fitBounds(bounds); // .setMaxBounds([[-360,-360],[360,360]])
				}
				globMap = map;
				
				// 关联背景地图资源
				L.tileLayer(url,{
					attribution: ''
				}).addTo(map);
				// 过滤后生成节点图层
				vm.createEnbLayer(vm.filterNodesByStatus(resObj.enb), globMap);
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
			},
			/**
			* 设备列表滑动检测
			* @param ev{event}: 鼠标事件
			**/
			observSlider(ev){// 通过事件代理，将逻辑抽离到父级，避免动态节点时，子节点的复杂绑定逻辑
				var vm = this,
					target = ev.target,
					clsList = target.classList;

				/* 切换当前行箭头状态 */
				if(clsList.contains('el-icon-arrow-right') || clsList.contains('el-icon-arrow-left')) {
					if(clsList.contains('el-icon-arrow-right')) {
						vm.showSlider(ev,true);
					}else {
						vm.showSlider(ev,false);
					}

					['el-icon-arrow-right','el-icon-arrow-left'].map(function(cls){
						if(clsList.contains(cls)) clsList.remove(cls);
						else clsList.add(cls);
					});

					/* 重置其他行箭头状态 */
					Array.from(document.querySelectorAll('.row-op')).map(function(item){
						if(item != target) {
							item.classList.remove('el-icon-arrow-left');
							item.classList.add('el-icon-arrow-right');
						}
					});
				}
			},
			/**
			* 设备列表滑动检测
			* @param evt{event}: 鼠标事件
			* @param bool{boolean}: 是否展开
			**/
			showSlider(evt,bool) {
				var vm = this,
					slider = vm.$refs.deviceSlider;
				
				/*根据事件对象调整位置*/
				if(evt) {
					var pos = vm.reposition(slider,evt.target),
						top = pos.top;

					slider.style.top = top+'px';
				}

				slider.classList.add('loading');
				if(bool) slider.classList.add('show');
				else slider.classList.remove('show');

				setTimeout(function(){
					slider.classList.remove('loading');
				},1000);

				vm.$refs.enblist.advanceQuery();
				vm.$refs.cpelist.advanceQuery();
				vm.enbQueryParams.search_text = '';
				vm.cpeQueryParams.search_text = '';
			},
			/* *
			 * 相对偏移值  -- 实现动态计算、精准移动目标
			 * @param target: 计算的目标对象
			 * @param reference: 计算的参照物
			 * */
			reposition(target, reference) {
				var docH = window.innerHeight || document.documentElement.clientHeight ||
					document.body.clientHeight,
					tRect = reference.getBoundingClientRect(),
					mRect = target.getBoundingClientRect();
				/**
				 * 根据相对视口的坐标偏移，计算相对位置的偏移量
				 *（top偏移：rect的top的坐标偏移差，left偏移：rect的left坐标偏移差）
				 **/
				var tTop = tRect.top,
					tLeft = tRect.left,
					mTop = mRect.top,
					mLeft = mRect.left,
					top = target.offsetTop + (tTop - mTop) - 10,
					left = target.offsetLeft + (tLeft - mLeft);
				var offsetD = tRect.y + tRect.height;

				if (docH - offsetD < mRect.height) {
					//top = top - mRect.height - tRect.height;
					top -= mRect.height - docH + offsetD + 30;
				}
				return {
					left: left,
					top: top
				};
			},
			// 全屏设置
			FullScreen() {
				var el = document.querySelector('#map');
				var isFullscreen = document.fullScreen || document.mozFullScreen || document.webkitIsFullScreen;
				if (!isFullscreen) { //进入全屏,多重短路表达式
					(el.requestFullscreen && el.requestFullscreen()) ||
					(el.mozRequestFullScreen && el.mozRequestFullScreen()) ||
					(el.webkitRequestFullscreen && el.webkitRequestFullscreen()) || (el.msRequestFullscreen && el.msRequestFullscreen());

				} else { //退出全屏,三目运算符
					document.exitFullscreen ? document.exitFullscreen() :
						document.mozCancelFullScreen ? document.mozCancelFullScreen() :
						document.webkitExitFullscreen ? document.webkitExitFullscreen() : '';
				}
			},
			/**
			* 刷新基站TOPO 下 统计信息，填充状态栏
			* @param ids{string}: 设备组Id集，多个以逗号分隔
			**/
			refresh_topo_Statistics(ids) {
				var params = {
					switch_status: true,
					isDual: true,
					isMonitor: true,
					group_id: ids
				};
				
				$.post("${ctx}/cell/cpeinfos/getCellStatusStatistics.action", params, function(data) {
					// 更新连接状态统计数据
					if(!data["connection_status"]){
						$("#topo_cell_statis .connStatusStatistics-gps").text("0/0");
					}else{
						$("#topo_cell_statis .connStatusStatistics-gps").text(data["connection_status"]);
					}
					// 更新MME状态统计数据
					if(!data["mme_status"]){
						$("#topo_cell_statis .mmeStatusStatistics-gps").text("0/0");
					}else{
						$("#topo_cell_statis .mmeStatusStatistics-gps").text(data["mme_status"]);
					}
					// 更新激活状态统计数据
					if(!data["op_state"]){
						$("#topo_cell_statis .opStatusStatistics-gps").text("0/0");
					}else{
						$("#topo_cell_statis .opStatusStatistics-gps").text(data["op_state"]);
					}
				}, "json");
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
		}
	});
	/**
	* 刷新topo图 -- 节点数据变动
	* @param obj{object}：信息有变动的节点
	**/
	function refreshTopo(obj) {
		var vm = topovm,
			latLonList = vm.allNodes.map(function(node){ return node.lat+'-'+node.lon; }),
			index = -1;
		
		vm.allNodes.map(function(item,idx){
			if(item.cellCode == obj.cellCode) {
				Object.assign(item, obj);
				index = idx;
			}
		});

		var latlonKey = obj.lat+'-'+obj.lon;
		if(vm.repeatedMap[latlonKey]) {
			vm.allNodes.splice(index,1);
			vm.repeatedMap[latlonKey].push(obj.code);
		}

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
	function motion(radius) {
		// step 时曲度：值越大越接近圆曲率
		var i = 0,j = step = 0.25;

		return function() {
			i += j;

			var r = radius*Math.pow(i,0.5),ang = 36;
			var x = r*Math.sin(i),y = r*Math.cos(i);

			i<0 && (j = step);
			i>ang && (j = -step);

			return [x, y];
		}
	}
	/**
	* enb数据规范化 -- {code: '',lat: '',lon: '',name: '',type: '',online: '',active: '', ... }为必要属性
	* @param nodeList{array}：enb节点原始数据
	**/
	function transformNode(nodeList){
		var rds = motion(0.0001);
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
				
				var pointer =  rds();
				lonr = pointer[0];
				latr = pointer[1];
				
				lon = lon-0 + lonr + '';
				lat = lat-0 + latr + '';
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
		var ctn = $(dom).parents('.leaflet-marker-icon');

		$('.top-z-index').removeClass('top-z-index');
		ctn.addClass('top-z-index');
		
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
				layerOptions: {
					draggable: true,
					dragend: function(evt){
						var target = evt.target,
							latlng = target._latlng
							olatlng = target._olatlng,
							icon = target._icon,
							sn = $('.node-code',icon).text()
							code = $('.cell-code',icon).text();

						if([undefined,'undefined'].includes(code)){// 拖拽不可设置gps部分
							target.setLatLng([olatlng.lat,olatlng.lng]);
							return;
						}
						// 拖拽节点gps设置确认
						$.messager.confirm({
							title: '<%=rb.getString("QueRen")%>',
							msg: '<%=rb.getString("ChongSheGPS")%>',
							fn: function(r){
								if(r){
									$.ajax({
										url: '${ctx}/cell/topo/setLocationInfo.action',
										type: 'post',
										dataType: 'json',
										data: {
											longitude: latlng.lng.toFixed(3),
											latitude: latlng.lat.toFixed(3),
											cell_code: code,
											operator_code: operator_code
										},
										success: function(data){
											if(data['success']){
												var vm = topovm,
													latlonKey = olatlng.lat+'-'+olatlng.lng,
													repeatCodes = vm.repeatedMap[latlonKey];
													
												if(repeatCodes && repeatCodes.length>1){
													var topCode = repeatCodes.splice(1,1)[0];

													vm.repeatedHideNodes.map(function(item){
														if(item.code == topCode) {
															vm.allNodes.push(item);
															vm.prevNodes.push(item);
														}
													});
												}

												refreshTopo({
													cellCode: code,
													lat: latlng.lat.toFixed(3),
													lon: latlng.lng.toFixed(3),
													olat: latlng.lat.toFixed(3),
													olon: latlng.lng.toFixed(3)
												});
												topovm.$refs.enbDevice.refresh();
												topovm.$refs.cpeDevice.refresh();
											}else{
												toast(data['message'],$('#omc_app_ctn'));
											}
										}
									})
								}else{
									target.setLatLng([olatlng.lat,olatlng.lng]);
								}
							},
							onClose: function(){
								target.setLatLng([olatlng.lat,olatlng.lng]);
							}
						}).addClass("seriousConfirm");
					},
					fill: false,
					stroke: false,
					weight: 0,
					color: '#A0A0A0'
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
						$('.repeated-list').hide();
					});
					
					$(window).resize();
					// 点击节点弹出明细层
					layer.bindPopup($(showNodeInfo(record)).wrap('<div/>').parent().html(),{autoPan: false, keepInView: true});
				}
			};
        return options;
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

