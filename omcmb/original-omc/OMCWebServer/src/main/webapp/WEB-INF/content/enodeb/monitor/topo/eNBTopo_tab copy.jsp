<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<link rel="stylesheet" href="${ctx}/js/topo/leaflet.css" type="text/css" media="screen" />
<link rel="stylesheet" href="${ctx}/js/topo/dvf.css" type="text/css" media="screen"/>
<link rel="stylesheet" href="${ctx}/js/topo/node.css" type="text/css" media="screen"/>
<style>
	.width-mini .el-input.el-input--small {
		width: 80% !important;
	}
	.width-mini .advanceQuery,.no-margin .advanceQuery {
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
		position: absolute; 
		top: 40px;
		left: -800px;
		width: 520px;
		height: 100%; 
		background-color: #fff;
		box-shadow: 2px 3px 8px #cdc6c6;
		transition: left 0.5s ease;
	}
	.device-slider.show {
		left: 341px;
		z-index: 100;
		top: 0px;
		min-height: 96%;
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
		text-align: left;
		color: #B0B0B0;
		white-space: nowrap;
		padding: 2px 5px;
	}
	.node-ul {
		padding-top: 5px;
	}
	.node-ul li {
		padding: 2px;
		display: flex;
		align-items: center;
	}
	.alarm-info {
		right: -45px;
	}
	.node-item-leaf .el-icon-topo-enb::before, 
	.node-item-leaf .el-icon-topo-cpe::before, 
	.el-icon-topo-enb.online::before,
	.el-icon.online::before {
		color: #40BC33;
	}
	.node-item-leaf.offline .el-icon-topo-enb::before,
	.node-item-leaf.offline .el-icon-topo-cpe::before,
	.el-icon.offline::before {
		color: #999;
	}
	.el-icon.inactive::before {
		color: #E88282;
	}
	.el-icon-topo-enb.registed::before,.el-icon-topo-cpe.registed::before,
	.registed .el-icon-topo-enb::before,.registed .el-icon-topo-cpe::before {
		color: #F99C9C !important;
	}
	.el-icon-topo-enb.granted::before,.el-icon-topo-cpe.granted::before,
	.granted .el-icon-topo-enb::before,.granted .el-icon-topo-cpe::before {
		color: #F8D675 !important;
	}
	.el-icon-topo-enb.authed::before,.el-icon-topo-cpe.authed::before,
	.authed .el-icon-topo-enb::before,.authed .el-icon-topo-cpe::before {
		color: #88D46D !important;
	}
	.leaflet-popup-content {
		margin: 5px !important;
		overflow: hidden;
	}
	.leaflet-popup-content-wrapper {
		border: none;
		border-radius: 5px;
	}
	.leaflet-popup-content .tabsTitle, .device-slider .el-tabs__header {
		background: #fff;
	}
	.leaflet-popup-content .tabsContentDiv {
		border-width: 0px;
		top: 0px;
	}
	.leaflet-popup.leaflet-zoom-animated {
		bottom: 15px !important;
	}
	.leaflet-popup-content .tabsTitle {
		color: 363B4E;
	}
	.node-info-close {
		position: absolute;
		top: 0px;
		right: 0px;
		z-index: 1;
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

	.repeated-nodes {
		display: inline-block;
		width: 32px;
		height: 30px;
		position: absolute;
		top: -1px;
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
		padding: 0px 3px;
		position: absolute;
		left: 25px;
		top: 23px;
		display: inline-block;
		height: 13px;
		line-height: 15px;
		min-width: 20px;
		text-align: center;
		background-color: #fff;
		border-radius: 7px;
		border: 1px solid #5c5cf0;
	}
	.top-z-index {
		z-index: 99999 !important;
	}

	.topo-legend-cls {
		padding: 80px 15px 10px 15px;
		position: absolute; 
		right: 15px; 
		top: 50px;
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
		color: #363b4e;
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
		background-color: #324044;
		font-weight:bold;
		font-size:14px;
		color: #fff;
		border-radius: 5px 5px 0 0;
	}

	.info-panel {
		position: absolute;
		top: 34px;
		right: -341px;
		height: calc(100% - 35px);
		width: 340px;
		background-color: #fff;
		z-index: 1000;
		box-shadow: 2px 2px 10px rgba(178,219,255,0.3);
		transition: right 0.5s ease;
	}
	.info-panel.show,.site-panel.show {
		right: 0px;
	}
	.info-panel-title {
		padding: 10px;
		font-weight: bold;
		border-bottom: 1px solid #e9e9e9;
	}
	.info-item-cls {
		padding: 8px 5px 8px 20px;
		display: flex;
		align-items: center;
	}
	.info-item-cls::before {
		display: inline-block;
		content: attr(label);
		width: 100px;
		color: #666;
	}
	
	.panel-arrow {
		padding: 3px 0 3px 6px;
		position: absolute;
		min-width: 20px !important;
		max-width: 20px !important;
		max-height: 18px !important;
		top: calc(50% + 10px);
		right: 314px;
		z-index: 100;
		background-color: #4D84FF !important;
		border-radius: 15px 0 0 15px;
		box-shadow: 2px 0 15px rgba(0,0,0,.2);
		transition: right 0.5s ease;
		transform: rotate(180deg);
	}
	
	.panel-arrow i {
		font-size: 12px;
		transform: rotate(90deg);
	}
	.expanded .panel-arrow i {
		transform: rotate(-90deg);
	}
	.panel-arrow i::before {
		color: #fff !important;
	}

	.short-query  .pairgrid-query {
		width: 180px;
	}
	.margin-right-5 {
		margin-right: 5px;
	}

	.margin-right-5.el-icon-menu-alarm::before {
		font-size: 16px;
	}
	.Critical.el-icon-menu-alarm::before {
		color: #ff7b7b;
	}
	.Major.el-icon-menu-alarm::before {
		color: #fb9f50;
	}
	.Minor.el-icon-menu-alarm::before {
		color: #d0d53b;
	}
	.Warning.el-icon-menu-alarm::before {
		color: #67dff8;
	}

	.node-item-leaf.selected::before {
		position: absolute;
		content: '';
		display: inline-block;
		padding: 16px;
		border-radius: 16px;
		background-color: #fff;
		opacity: 0.7;
		z-index: -1;
		top: -2px;
	}
	.node-item-leaf.selected::after {
		position: absolute;
		content: '';
		display: inline-block;
		padding: 25px;
		border-radius: 25px;
		background-color: #fff;
		opacity: 0.35;
		z-index: -2;
		top: -11px;
	}

	.node-item-leaf.site-selected::before {
		position: absolute;
		content: '';
		display: inline-block;
		padding: 16px;
		border-radius: 16px;
		background-color: #fff;
		opacity: 0.7;
		z-index: -1;
		top: 6px;
	}
	.node-item-leaf.site-selected::after {
		position: absolute;
		content: '';
		display: inline-block;
		padding: 25px;
		border-radius: 25px;
		background-color: #fff;
		opacity: 0.35;
		z-index: -2;
		top: -3px;
	}

	.leaflet-popup-tip {
		width: 5px;
		height: 5px;
		margin: -5px auto 0;
	}

	.el-icon-status-enable.disabled {
		color: gray;
	}

	.flex-bt-cls {
		display: flex;
		align-items: center;
		padding: 0 10px;
		border-left: 1px dashed #e9e9e9; 
	}
	.flex-bt-cls:hover {
		color: var(--main-color);
	}
	.flex-bt-cls.selected {
		color: var(--main-color);
	}
	.line-text {
		display: flex;
	}
	.white-space-nowrap {
		white-space: nowrap;
		margin-right: 5px;
	}
	.alarm-circle {
		display: inline-block;
		border-radius: 18px;
		width: 18px;
		height: 18px;
		line-height: 18px;
		text-align: center;
	}
	.Critical.alarm-circle {
		background-color: #ff7b7b;
		color: #fff;
	}
	.Major.alarm-circle {
		background-color: #fb9f50;
		color: #fff;
	}
	.Minor.alarm-circle {
		background-color: #d0d53b;
		color: #fff;
	}
	.Warning.alarm-circle {
		background-color: #67dff8;
		color: #fff;
	}
	.is-bottom-active .el-tabs__active-bar {
		top: auto;
	}
	.is-bottom-active .el-tabs__nav-scroll {
		border-bottom: 2px solid #E9E9E9;
	}
	.mme-list {
		position: relative;
		display: flex;
		max-width: 825px;
		flex-wrap: wrap;
		max-height: 500px;
		overflow: auto;
		padding:0 15px 15px 15px;
	}
	.mme-list-item {
		display: flex;
		padding: 10px;
		border: 1px solid #E9E9E9;
		margin:5px;
		border-radius:2px;
	}
	.mme-list-item .el-icon-status-MME {
		margin-top:20px;
	}
	.mme-info {
		display: flex;
		flex-direction: column;
		margin-left:5px;
	}
	.mmePopoverClass .el-popover__title{
		height:35px;
		line-height:35px;
		margin-bottom:0;
		padding:0 20px 0;
	}

	.site-pane-cls {
		padding: 10px;
		display: flex;
		align-items: center;
		justify-content: space-between;
		border-bottom: 1px solid #F3F3F3;
	}
	.site-panel {
		position: absolute;
		top: 0px;
		right: -401px;
		height: 100%;
		width: 400px;
		background-color: #fff;
		z-index: 1000;
		box-shadow: 2px 2px 10px rgba(178,219,255,0.3);
		transition: right 0.5s ease;
	}
	#topo_container .el-checkbox.is-bordered.el-checkbox--small {
		height: 28px;
	}
</style>

<style>
	.pick-model {
		z-index: -1 !important;
	}
	.detail-panel {
		width: 400px;
		position: absolute;
		top: 65px;
		right: 405px;
		background-color: #fff;
		z-index: 100;
	}
	.detail-header {
		padding: 10px;
		display: flex;
		justify-content: space-between;
		border-bottom: 1px solid #F3F3F3;
	}
	.detail-body {
		flex: auto;
		overflow: auto;
	}
	.info-pane-item {
		display: flex;
		align-items: center;
		padding: 5px 5px 5px 20px;
		margin-top: 5px;
	}
	.info-pane-item .item-title {
		width: 100px;
		color: rgba(0,0,0,.4);
		font-size: 12px;
	}
	.blue-bg::before {
		color: #2779F5;
	}
	.site-device-tag {
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 1px 2px;
		height: 22px;
		min-width: 30px;
		margin-top: -15px;
		border-radius: 3px;
		border: 1px solid #4D84FF;
		background-color: #fff;
		overflow: hidden;
	}
	.site-device-tag:empty {
		display: none;
	}
	.site-device-tag .el-icon::before {
		font-size: 22px;
	}
	.site-device-tag .el-icon_tpopo_2G {
		color: #FF8F1F;
	}
	.site-device-tag .el-icon_tpopo_5G {
		color: #08AC42;
	}
	.site-device-tag .el-icon-topo-enb::before {
		font-size: 25px;
		color: #4D84FF;
	}
	.space-between {
		justify-content: space-between;
		padding-right: 15px !important;
	}
	.no-padding-top .el-input {
		margin-top: 0px !important;
	}
	.width-fitcontent {
		width: 80%;
	}
	.width-fitcontent .advanceQuery {
		width: 100% !important;
		margin-left: 0px;
	}
	.width-fitcontent .el-input.el-input--small {
		width: calc(100% - 40px);
	}
	.no-margin .el-checkbox {
		margin-left: 0px;
		margin-bottom: 10px;
	}

	.all-device-wrap {
		position: absolute;
		top: 90px;
		right: 0;
		left: 0;
	}
	.all-device-wrap .el-table {
		margin: 0 auto;
		width: 50%;
		min-width: 880px;
		border-radius: 5px;
	}
	.all-device-wrap .el-table th {
		background-color: #324044;
		color: #fff;
	}

	.border-overview .el-table__header th {
		border-right: 1px solid #484747;
	}
</style>
<div class="overflow-cls">
	<div id="topo_container" style="height: 99.5%;width: 100%;min-width: 900px;overflow: hidden;display: flex;padding: 2px 0px 0px 5px;background: #fff;">
		<!-- Device group -->
		<div style="width: 340px;height: 100%;position: relative;" class="flex-ctn" v-show="groupShow">
			<div class="flex-ctn absolute-ctn">
				<div v-if="activeType == 'site'" style="position: absolute;z-index: 100;left: 310px;top: 6px;">
					<span @click="addSite" class="el-icon el-icon-operation-add"></span>
				</div>
				<el-tabs style="height: 100%;position: relative;" v-model="activeType" @tab-click="tabClick">
					<el-tab-pane label="<%=rb.getString("SheBeiZu")%>" name="group">
						<div style="flex: auto;" @click="observSlider">
							<el-ctable ref="group" :pagination="false" :rownumber="false" :query-params="groupQueryParams" :url="groupURL" @selection-change="groupChange" row-key="id"
								@load-success="loadSuccess">
								<template slot="toolbar">
									<el-query style="zoom: 0.8;" class="width-fitcontent" type="normal" @query="queryGroup" placeholder="<%=rb.getString("SheBeiZuMingCheng")%>"></el-query>
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
					</el-tab-pane>
					<el-tab-pane v-if="siteEnable" label='<%=rb.getString("ZhanDianMingCheng")%>' name="site">
						<div style="flex: auto;overflow: auto;">
							<el-ctable ref="siteList" :rownumber="false" row-key="id" height="100%"
								row-key="siteName"
								:url="siteListURL"
								:query-params="siteQuery"
								:show-pager="false"
								@row-dblclick="rowdblclickSite"
								@row-click="siteCurrentChange"
							>
								<template slot="toolbar">
									<el-query style="zoom: 0.8;" class="width-fitcontent" type="normal" @query="querySite" placeholder='<%=rb.getString("ZhanZhiMingCheng")%>'></el-query>
								</template>
								<el-table-column label='<%=rb.getString("ZhanZhiMingCheng")%>' prop="siteName"></el-table-column>
								<el-table-column label='<%=rb.getString("WeiDu")%>' width="75" prop="latitude"></el-table-column>
								<el-table-column label='<%=rb.getString("JingDu")%>' width="85" prop="longitude"></el-table-column>
								<el-table-column width="70">
									<template slot-scope="scope">
										<span @click="modifySite(scope.row, event)" class="el-icon el-icon-operation-edit" style="margin-right: 5px;"></span>
										<span @click="delSite(scope.row, event)" class="el-icon el-icon-operation-delete"></span>
									</template>
								</el-table-column>
							</el-ctable>
						</div>
					</el-tab-pane>
				</el-tabs>
			</div>
			<!-- device slider -->
			<div ref="deviceSlider" class="device-slider flex-ctn">
				<!-- device list -->
				<div style="flex: auto;overflow: auto;">
					<el-tabs style="height: 100%;" class="is-bottom-active">
						<el-tab-pane label="<%=rb.getString("XiaoZhan")%>">
							<el-ctable ref="enbDevice" @row-dblclick="rowdblclick" :rownumber="true" :url="deviceURL" :query-params="enbQueryParams" :front-pagination="true">
								<template slot="toolbar">
									<div style="display: flex; flex-wrap: wrap; align-items: center;padding: 0 10px;">
										<el-query style="zoom: 0.8;" ref="enblist" class="no-margin" type="normal" @query="queryDevice" placeholder="<%=rb.getString("SheBeiMingCheng")%>/<%=rb.getString("HostName")%>"></el-query>
										
										<div style="padding: 5px 0 0 0;zoom: 0.9;">
											<el-checkbox v-model="enbQueryParams.no_gps" true-label="" false-label="1"><%=rb.getString("WuJingWeiDuSheBei")%></el-checkbox>
											<span v-if="hasSAS" style="color: #a9a9a9;"><%=rb.getString("WuJingWeiDuSheBeiTiShi")%></span>
										</div>
									</div>
								</template>
								<el-table-column label="<%=rb.getString("XiaoZhanBianMa")%>" prop="serial_number" min-width="120"></el-table-column>
								<el-table-column label="<%=rb.getString("HostName")%>" prop="host_name"></el-table-column>
								<el-table-column label="<%=rb.getString("WeiDu")%>" prop="latitude"></el-table-column>
								<el-table-column label="<%=rb.getString("JingDu")%>" prop="longitude"></el-table-column>
								<el-table-column width="45">
									<template slot-scope="scope">
										<div v-if="!sasEnable" class="el-icon el-icon-operation-settings" @click="optClick(scope.row,event)"></div>
										<div v-if="sasEnable" class="el-icon el-icon-operation-settings disabled"></div>
									</template>
								</el-table-column>
							</el-ctable>
						</el-tab-pane>
						
						<el-tab-pane label="<%=rb.getString("CPE")%>" v-if="writableMap['CODE_CPE_MONITOR'] != undefined">
							<el-ctable ref="cpeDevice" @row-dblclick="rowdblclick" :rownumber="true" :url="cpeDeviceURL" :query-params="cpeQueryParams" :front-pagination="true">
								<template slot="toolbar">
									<div style="display: flex; flex-wrap: wrap; align-items: center;padding: 0 10px;">
										<el-query style="zoom: 0.8;" ref="cpelist" class="no-margin" type="normal" @query="queryCpeDevice" placeholder="<%=rb.getString("CPEXuLieHao")%>/<%=rb.getString("CPEName")%>"></el-query>
										
										<div style="padding: 5px 0 0 0;zoom: 0.9;">
											<el-checkbox v-model="cpeQueryParams.no_gps" true-label="" false-label="1"><%=rb.getString("WuJingWeiDuSheBei")%></el-checkbox>
											<span v-if="hasSAS" style="color: #a9a9a9;"><%=rb.getString("WuJingWeiDuSheBeiTiShi")%></span>
										</div>
									</div>
								</template>
								<el-table-column label="<%=rb.getString("CPEXuLieHao")%>" prop="serial_number" min-width="130"></el-table-column>
								<el-table-column label="<%=rb.getString("CPEName")%>" prop="host_name" min-width="100"></el-table-column>
								<el-table-column label="<%=rb.getString("WeiDu")%>" prop="latitude" min-width="90"></el-table-column>
								<el-table-column label="<%=rb.getString("JingDu")%>" prop="longitude" min-width="90"></el-table-column>
								<el-table-column width="45">
									<template slot-scope="scope">
										<div v-if="!sasEnable" class="el-icon el-icon-operation-settings" @click="optClick(scope.row,event)"></div>
										<div v-if="sasEnable" class="el-icon el-icon-operation-settings disabled"></div>
									</template>
								</el-table-column>
							</el-ctable>
						</el-tab-pane>
					</el-tabs>
				</div>
			</div>
		</div>
		<!-- topo map -->
		<div v-show="activeType=='group'" style="flex: auto; display: flex; flex-direction: column;position: relative;background: #fff;">
			<div :class="arrowClass" @click="groupShow = !groupShow"></div>
			<!-- 操作按钮 -->
			<div style="min-height: 30px;">
				<div style="padding: 8px;overflow: hidden;text-overflow:ellipsis;white-space:nowrap;width: 300px;color: #363B4E;font-size: 14px;">{{groupNames}}</div>
			</div>
			<div style="position: absolute; top: 50px; right: 180px;padding: 5px;background: #fff;z-index: 1;border-radius: 5px;display:flex;">
				<el-popover :append-to-body="false">
					<div slot="reference" class="flex-bt-cls">
						<i class="el-icon el-icon-operation-settings margin-right-5"></i> <%=rb.getString("SheZhi")%>
					</div>
					<el-form style="width: 500px;padding: 15px;" class="form-item-bottom-15">
						<el-form-item>
							<el-checkbox-group v-model="statusForm.deviceType" style="margin-top: 10px;" @change="setStatus">
								<el-checkbox disabled label="enb"><%=rb.getString("XiaoZhan")%></el-checkbox>
								<el-checkbox v-if="cpeEnable" label="cpe"><%=rb.getString("CPE")%></el-checkbox>
							</el-checkbox-group>
						</el-form-item>
						<el-form-item label="">
							<div>
								<%=rb.getString("SheBeiZaiXianZhuangTai")%>
								<el-radio v-show="false" v-model="statusType" label="deviceStatus" @change="statusTypeChange('deviceStatus')"><%=rb.getString("SheBeiZaiXianZhuangTai")%></el-radio>
							</div>
							<el-checkbox-group v-model="statusForm.deviceStatus" style="margin-left: 25px;margin-top: 10px;" @change="statusChange('deviceStatus')">
								<el-checkbox label="on"><%=rb.getString("ZaiXian")%></el-checkbox>
								<el-checkbox label="off"><%=rb.getString("LiXian")%></el-checkbox>
							</el-checkbox-group>
						</el-form-item>

						<el-form-item label="">
							<div>
								<%=rb.getString("ShiFouJiHuo")%>
								<el-radio v-show="false" v-model="statusType" label="activeStatus" @change="statusTypeChange('activeStatus')"><%=rb.getString("ShiFouJiHuo")%></el-radio>
							</div>
							<el-checkbox-group v-model="statusForm.activeStatus" style="margin-left: 25px;margin-top: 10px;" @change="statusChange('activeStatus')">
								<el-checkbox label="yes"><%=rb.getString("JiHuo")%></el-checkbox>
								<el-checkbox label="no"><%=rb.getString("QuJiHuo")%></el-checkbox>
							</el-checkbox-group>
						</el-form-item>

						<el-form-item label="" v-show="hasSAS">
							<div>
								<%=rb.getString("SASZhuangTaiQuanCheng")%>
								<el-radio v-show="false" v-model="statusType" label="sasStatus" @change="statusTypeChange('sasStatus')"><%=rb.getString("SASZhuangTaiQuanCheng")%></el-radio>
							</div>
							<el-checkbox-group v-model="statusForm.sasStatus" style="margin-left: 25px;margin-top: 10px;" @change="statusChange('sasStatus')">
								<el-checkbox label="Unregistered">Unregistered</el-checkbox>
								<el-checkbox label="Registered">Registered</el-checkbox>
								<el-checkbox label="Granted">Granted</el-checkbox>
								<el-checkbox label="Authorized">Authorized</el-checkbox>
							</el-checkbox-group>
						</el-form-item>
					</el-form>
				</el-popover>
				<div class="flex-bt-cls" @click="measure">
					<i class="el-icon el-icon-ranging margin-right-5"></i> <%=rb.getString("CeJu")%>
				</div>
				<div class="flex-bt-cls" v-show="isLocal" @click="FullScreen">
					<i class="el-icon el-icon-fullscreen margin-right-5"></i> <%=rb.getString("QuanPin")%>
				</div>
				<div class="flex-bt-cls" @click="refreshTopoData">
					<i class="el-icon el-icon-common-refresh margin-right-5"></i><%=rb.getString("ShuaXin")%>
				</div>
			</div>
			<!-- 图例 -->
			<div class="topo-legend-cls" v-if="!topoOverViewEnable && statusType=='deviceStatus'">
				<div class="legend-title location-title"><%=rb.getString("WeiZhi")%></div>
				<div class="legend-title" style="margin-top: 25px;">
					<div style="font-weight: bold;text-align: left;padding-left: 15px;">
						<%=rb.getString("XiaoZhan")%> <%=rb.getString("SheBeiZongShu")%>：{{deviceInfos.enbTotal}} <br/>
						<span v-if="cpeEnable"><%=rb.getString("CPE")%> <%=rb.getString("SheBeiZongShu")%>：{{deviceInfos.cpeTotal}}</span>
					</div>
				</div>
				<div><i class="el-icon el-icon-topo-enb online"></i> <%=rb.getString("XiaoZhan")%> - <%=rb.getString("ZaiXian")%> ({{deviceInfos.enbOnlineCount}})</div>
				<div><i class="el-icon el-icon-topo-enb offline"></i> <%=rb.getString("XiaoZhan")%> - <%=rb.getString("LiXian")%> ({{deviceInfos.enbTotal - deviceInfos.enbOnlineCount}})</div>
				<div v-if="cpeEnable"><i class="el-icon el-icon-topo-cpe online"></i> <%=rb.getString("CPE")%> - <%=rb.getString("ZaiXian")%> ({{deviceInfos.cpeOnlineCount}})</div>
				<div v-if="cpeEnable"><i class="el-icon el-icon-topo-cpe offline"></i> <%=rb.getString("CPE")%> - <%=rb.getString("LiXian")%> ({{deviceInfos.cpeTotal - deviceInfos.cpeOnlineCount}})</div>
			</div>
			<!-- 图例 -->
			<div class="topo-legend-cls" v-if="!topoOverViewEnable && statusType=='activeStatus'">
				<div class="legend-title location-title"><%=rb.getString("WeiZhi")%></div>
				<div class="legend-title" style="margin-top: 25px;">
					<div style="font-weight: bold;text-align: left;padding-left: 15px;">
						<%=rb.getString("XiaoZhan")%> <%=rb.getString("SheBeiZongShu")%>：{{deviceInfos.enbTotal}} <br/>
						<span v-if="cpeEnable"><%=rb.getString("CPE")%> <%=rb.getString("SheBeiZongShu")%>：{{deviceInfos.cpeTotal}}</span>
					</div>
				</div>
				<div><i class="el-icon el-icon-topo-enb online"></i> <%=rb.getString("XiaoZhan")%> - <%=rb.getString("JiHuo")%> ({{deviceInfos.enbActiveCount}})</div>
				<div><i class="el-icon el-icon-topo-enb offline"></i> <%=rb.getString("XiaoZhan")%> - <%=rb.getString("QuJiHuo")%> ({{deviceInfos.enbTotal - deviceInfos.enbActiveCount}})</div>
			</div>
			<!-- 图例 -->
			<div class="topo-legend-cls" v-if="!topoOverViewEnable && statusType=='sasStatus'">
				<div class="legend-title location-title"><%=rb.getString("WeiZhi")%></div>
				<div class="legend-title" style="margin-top: 25px;">
					<div style="font-weight: bold;text-align: left;padding-left: 15px;">
						<%=rb.getString("XiaoZhan")%> <%=rb.getString("SheBeiZongShu")%>：{{deviceInfos.enbTotal}} <br/>
						<span v-if="cpeEnable"><%=rb.getString("CPE")%> <%=rb.getString("SheBeiZongShu")%>：{{deviceInfos.cpeTotal}}</span>
					</div>
				</div>
				<div>
					Unregistered ({{deviceInfos.unregistered}})
					<div style="padding: 8px 0;border-bottom: 1px dashed #e3e3e3;">
						<i class="el-icon el-icon-topo-enb offline"></i> <%=rb.getString("XiaoZhan")%>({{deviceInfos.enbunregistered}})
						<span v-if="cpeEnable"><i class="el-icon el-icon-topo-cpe offline"></i> <%=rb.getString("CPE")%>({{deviceInfos.unregistered - deviceInfos.enbunregistered}})</span>
					</div>
				</div>
				<div>
					Registered ({{deviceInfos.registered}})
					<div style="padding: 8px 0;border-bottom: 1px dashed #e3e3e3;">
						<i class="el-icon el-icon-topo-enb registed"></i> <%=rb.getString("XiaoZhan")%>({{deviceInfos.enbregistered}})
						<span v-if="cpeEnable"><i class="el-icon el-icon-topo-cpe registed"></i> <%=rb.getString("CPE")%>({{deviceInfos.registered - deviceInfos.enbregistered}})</span>
					</div>
				</div>
				<div>
					Granted ({{deviceInfos.granted}})
					<div style="padding: 8px 0;border-bottom: 1px dashed #e3e3e3;">
						<i class="el-icon el-icon-topo-enb granted"></i> <%=rb.getString("XiaoZhan")%>({{deviceInfos.enbgranted}})
						<span v-if="cpeEnable"><i class="el-icon el-icon-topo-cpe granted"></i> <%=rb.getString("CPE")%>({{deviceInfos.granted - deviceInfos.enbgranted}})</span>
					</div>
				</div>
				<div>
					Authorized ({{deviceInfos.authorized}})
					<div style="padding: 8px 0;">
						<i class="el-icon el-icon-topo-enb authed"></i> <%=rb.getString("XiaoZhan")%>({{deviceInfos.enbauthorized}})
						<span v-if="cpeEnable"><i class="el-icon el-icon-topo-cpe authed"></i> <%=rb.getString("CPE")%>({{deviceInfos.authorized - deviceInfos.enbauthorized}})</span>
					</div>
				</div>
			</div>
			<!-- topo 地图 -->
			<div style="flex: auto;overflow: auto;" :class="sasSwitchCls">
				<div id="map" style="width: 100%;min-height: 100%;background: #A1D4E0;"></div>
			</div>
			<!-- 状态数值 -->
			<div id="topo_cell_statis" style="height: 30px; line-height: 30px;text-align: right;padding-right: 10px;display: none;">
				<span><%=rb.getString("LianJieZhuangTai")%><%=rb.getString("MaoHao")%></span>
				<span class="connStatusStatistics-gps"></span>
			
				<span style="margin-left: 50px;"><%=rb.getString("ShiFouJiHuo")%><%=rb.getString("MaoHao")%></span>
				<span class="opStatusStatistics-gps"></span>
			
			<span style="margin-left: 50px;"><%=rb.getString("MMEZhuangTai")%><%=rb.getString("MaoHao")%></span>
			<span class="mmeStatusStatistics-gps"></span>
			</div>

			<!-- Zed 大屏 -->
			<div v-if="topoOverViewEnable" class="all-device-wrap">
				<el-table :data="overViewList" border stripe class="border-overview">
					<el-table-column type="index" width="30"></el-table-column>
					<el-table-column label="Area" prop="area"></el-table-column>
					<el-table-column label="BTS Total" prop="btsTotal"></el-table-column>
					<el-table-column label="BTS Online" prop="btsOnline"></el-table-column>
					<el-table-column label="eNB Total" prop="enbTotal"></el-table-column>
					<el-table-column label="eNB Online" prop="enbOnline"></el-table-column>
					<el-table-column label="gNB Total" prop="gnbTotal"></el-table-column>
					<el-table-column label="gNB Online" prop="gnbOnline"></el-table-column>
					<el-table-column label="Online Total" prop="onlineTotal"></el-table-column>
					<el-table-column label="Total" prop="total"></el-table-column>
				</el-table>
			</div>

			<!-- 信息区域 -->
			<div :class="infoPanelCls">
				<div class="panel-arrow" @click="hidePanel">
					<i class="el-icon el-icon-down"></i>
				</div>
				<div v-show="infoShow && infoForm.type=='enb'">
					<div class="info-panel-title"><%=rb.getString("XinXi")%></div>
					<div class="info-item-cls" label="<%=rb.getString("XiaoZhanBianMa")%>" style="margin-top: 10px;">{{infoForm.code}}</div>
					<div class="info-item-cls" label="<%=rb.getString("HostName")%>">{{infoForm.name}}</div>
					<div class="info-item-cls" label="<%=rb.getString("IPDiZhi")%>">{{infoForm.ip}}</div>
					<div class="info-item-cls" label="<%=rb.getString("SheBeiZu")%>">{{infoForm.groupName}}</div>
					<div class="info-item-cls" label="<%=rb.getString("ShiFouJiHuo")%>" v-html="activeFmt(infoForm.active, infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("GPSWeiZhi")%>">
						{{infoForm.lat}}, {{infoForm.lon}} 
						<span class="el-icon el-icon-operation-edit" v-show="!sasEnable" style="margin-left: 5px;" @click="modifyInfoGPS"></span>
					</div>
					<div class="info-item-cls" label="<%=rb.getString("GaoJingJiBie")%>" v-html="alarmFmt(infoForm.alarm,infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("MMEZhuangTai")%>" style="position: relative;">
						<div v-if="['','NULL','null',null,undefined].includes(infoForm.NEW_MME_STATUS)">
							<span v-html="oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'left')" ></span>
                            <el-popover title="All MME" popper-class="mmePopoverClass" v-if="oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'list').length > 0" :append-to-body="false">
                                <span style="color:#4d84ff;cursor:pointer;" slot="reference" v-if="oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'list').length > 0">
                                    [ <span v-html="oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'right')"></span> ]
                                </span>
                                <div class="mme-list">
                                    <i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
                                    <div class="mme-list-item" v-for="item in oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'list')">
                                        <span v-if="item.status == '1'" style="margin-top:10px;" class="el-icon el-icon-status-MME greenIcon"></span>
                                        <span v-if="item.status == '0'" style="margin-top:10px;" class="el-icon el-icon-status-MME redIcon"></span> 
                                        <div class="mme-info">
                                            <span>MME IP : {{item.mmeIp}}</span>
                                            <span v-if="item.status==='0'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEWeiLianJie")%></span>
                                            <span v-if="item.status=='1'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEYiLianJie")%></span>
                                            <span v-if="item.status=='2'"><%=rb.getString("MMEZhuangTai")%> : --</span>
                                            <span v-if="item.status==='' || item.status===null"><%=rb.getString("MMEZhuangTai")%> : </span>
                                        </div>
                                    </div>
                                </div>
                            </el-popover>
						</div>
						<div v-if="!['','NULL','null',null,undefined].includes(infoForm.NEW_MME_STATUS)" style="display: flex;align-items: center;">
							<span v-html="mmesStatusFmt(infoForm.NEW_MME_STATUS)" ></span>
							<el-popover title="All MME" popper-class="mmePopoverClass" :append-to-body="false">
                                <span style="color:#4d84ff;cursor:pointer;" slot="reference">
                                    [ {{parseMMEOnNum(infoForm.NEW_MME_STATUS)}}/{{parseMME(infoForm.NEW_MME_STATUS).length}} ]
                                </span>
                                <div class="mme-list">
                                    <i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
                                    <div class="mme-list-item" v-for="item in parseMME(infoForm.NEW_MME_STATUS)">
                                        <span v-if="item.status == '1'" class="el-icon el-icon-status-MME greenIcon"></span>
                                        <span v-if="item.status == '0'" class="el-icon el-icon-status-MME redIcon"></span> 
                                        <div class="mme-info">
                                            <span>MME IP : {{item.mmeIp}}</span>
                                            <span v-if="item.status==='0'"><%=rb.getString("MMEZhuangTai")%> : Disconnected</span>
                                            <span v-if="item.status=='1'"><%=rb.getString("MMEZhuangTai")%> : Connected</span>
                                            <span v-if="item.status=='2'"><%=rb.getString("MMEZhuangTai")%> : --</span>
                                            <span v-if="item.status==='' || item.status===null"><%=rb.getString("MMEZhuangTai")%> : </span>
                                            <span><%=rb.getString("PLMN")%> : {{item.plmnId}}</span>
                                        </div>
                                    </div>
                                </div>
                            </el-popover>
						</div>
					</div>
					<div class="info-item-cls" label="<%=rb.getString("RSMME")%>" v-html="mmeIPfmt(infoForm.mmeStatus,infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("SASKaiGuan")%>" v-show="hasSAS" v-html="sasEnableFmt(infoForm.mmeStatus,infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("SASZhuangTai")%>" v-show="hasSAS" v-html="sasStateFmt(infoForm.sasState,infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("PCI2")%>">{{infoForm.pci}}</div>
					<div class="info-item-cls" label="<%=rb.getString("ZuiDaBanJin")%>">{{infoForm.radius}}</div>
					<div class="info-item-cls" label="<%=rb.getString("ZuiXiaoBanJin")%>">{{infoForm.minRadius}}</div>
				</div>
				<div v-show="infoShow && infoForm.type=='cpe'">
					<div class="info-panel-title"><%=rb.getString("XinXi")%></div>
					<div class="info-item-cls" label="CPE SN" style="margin-top: 10px;">{{infoForm.sn}}</div>
					<div class="info-item-cls" label="eNB SN">{{infoForm.relaEnb}}</div>
					<div class="info-item-cls" label="<%=rb.getString("CPEName")%>" style="word-break: break-all;align-items: baseline;">{{infoForm.name}}</div>
					<div class="info-item-cls" label="<%=rb.getString("IMSI")%>">{{infoForm.imsi}}</div>
					<div class="info-item-cls" label="<%=rb.getString("IPDiZhi")%>">{{infoForm.ip}}</div>
					<div class="info-item-cls" label="<%=rb.getString("MACDiZhi")%>">{{infoForm.mac}}</div>
					<div class="info-item-cls" label="<%=rb.getString("GPSWeiZhi")%>">
						{{infoForm.lat}}, {{infoForm.lon}}
						<span class="el-icon el-icon-operation-edit" v-show="!sasEnable" style="margin-left: 5px;" @click="modifyInfoGPS"></span>
					</div>
					<div class="info-item-cls" label="<%=rb.getString("GaoJingJiBie")%>" v-html="alarmFmt(infoForm.alarm,infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("WeiXingXinHao")%>" style="position: relative;" v-html="rsrpFmt(infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("SheBeiZu")%>">{{infoForm.groupName}}</div>
					<div class="info-item-cls" label="<%=rb.getString("SASKaiGuan")%>" v-show="hasSAS" v-html="sasEnableFmt(infoForm.sasEnable, infoForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("SASZhuangTai")%>" v-show="hasSAS" v-html="sasStateFmt(infoForm.sasState, infoForm)"></div>
				</div>
				<!-- 重叠设备 -->
				<div v-show="repeatShow">
					<div class="info-panel-title"><%=rb.getString("SheBeiLieBiao")%></div>
					<el-ctable :data="repeatedTbData" row-key="cellCode" :pagination="false" height="90%">
						<template slot="toolbar">
							<el-query type="normal" class="short-query" @query="queryRepeated" placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%>"></el-query>
						</template>
						<el-table-column label="<%=rb.getString("XiaoZhanBianMa")%>" prop="code" width="150">
							<template slot-scope="scope">
								<div v-if="scope.row.sn">{{scope.row.sn}}</div>
								<div v-else>{{scope.row.code}}</div>
							</template>
						</el-table-column>
						<el-table-column label="<%=rb.getString("HostName")%>" prop="name" width="100"></el-table-column>
						<el-table-column label="" width="40">
							<template slot-scope="scope">
								<div class="el-icon el-icon-moveTop" :class="{disabled: scope.$index==0}" @click="moveTop(scope.row, scope.$index)"></div>
							</template>
						</el-table-column>
					</el-ctable>
				</div>
			</div>
		</div>

		<!-- Site Map -->
		<div v-show="activeType=='site'" style="flex: auto; display: flex; flex-direction: column;position: relative;background: #fff;">
			<!-- Site 地图 -->
			<div style="flex: auto;overflow: auto;">
				<div id="site_map" style="width: 100%;min-height: 100%;background: #A1D4E0;"></div>
			</div>

			<div style="position: absolute; top: 50px; right: 180px;padding: 5px;background: #fff;z-index: 1;border-radius: 5px;display:flex;">
				<el-popover :append-to-body="false">
					<div slot="reference" class="flex-bt-cls">
						<i class="el-icon el-icon-operation-settings margin-right-5"></i> <%=rb.getString("SheZhi")%>
					</div>
					<div style="padding: 15px; border-bottom: 1px solid #E9EdF9;">
						<el-checkbox :indeterminate="isIndeterminate" v-model="siteCheckAll" @change="siteCheckAllChange">All</el-checkbox>
					</div>
					<el-form style="padding: 0 15px;width: 100px;" class="form-item-bottom-15">
						<el-form-item>
							<el-checkbox-group v-model="siteDeviceType" @change="siteCheckChange" class="no-margin" style="margin-top: 10px;" @change="setStatus">
								<el-checkbox v-if="isGSMEnable" label="gsm">GSM</el-checkbox>
								<el-checkbox label="enb"><%=rb.getString("XiaoZhan")%></el-checkbox>
								<el-checkbox label="gnb"><%=rb.getString("gNB")%></el-checkbox>
							</el-checkbox-group>
						</el-form-item>
					</el-form>
					<div v-if="false" style="padding: 10px 15px;">
						<el-button type="primary" size="mini" style="min-width:60px;"><%=rb.getString("QueDing")%></el-button>
						<el-button size="mini" style="min-width:60px;padding: 0 10px;"><%=rb.getString("QuXiao")%></el-button>
					</div>
				</el-popover>
				<div class="flex-bt-cls" @click="measure">
					<i class="el-icon el-icon-ranging margin-right-5"></i> <%=rb.getString("CeJu")%>
				</div>
				<div class="flex-bt-cls" v-show="isLocal" @click="FullScreen">
					<i class="el-icon el-icon-fullscreen margin-right-5"></i> <%=rb.getString("QuanPin")%>
				</div>
				<div class="flex-bt-cls" @click="getSiteList">
					<i class="el-icon el-icon-common-refresh margin-right-5"></i><%=rb.getString("ShuaXin")%>
				</div>
			</div>

			<!-- 图例 -->
			<div class="topo-legend-cls" v-if="activeType=='site'" style="padding: 49px 15px 10px 15px;">
				<div class="legend-title location-title"><%=rb.getString("WeiZhi")%></div>
				<div class="legend-title" style="margin-top: 25px;">
					<div style="font-weight: bold;text-align: left;padding-left: 15px;">
						<%=rb.getString("ZhanDianMingCheng")%> <%=rb.getString("SheBeiZongShu")%>：{{allSiteLatlonNodes.length}} <br/>
					</div>
				</div>
			</div>

			<!-- 信息区域 -->
			<div :class="sitePanelCls" style="height: 100%;top: 0px;display: flex;flex-direction: column;">
				<div class="site-pane-cls">
					<span v-if="siteType == 'view'" style="font-weight: bold;color: #7A7992;">{{currentSiteRow.siteName}}</span>
					<span v-if="siteType == 'add'" style="font-weight: bold;color: #7A7992;"><%=rb.getString("TianJia")%></span>
					<span style="display: flex;align-items: center;">
						<i v-if="siteType == 'view'" @click="toAddSiteDevice" class="el-icon el-icon-operation-add" style="margin-right: 10px;"></i>
						<i @click="siteInfoShow = false" class="el-icon el-icon-close"></i>
					</span>
				</div>

				<!-- add Site -->
				<div v-show="siteType == 'add'" style="padding: 15px;flex: auto;overflow: auto;">
					<el-form ref="addForm" :model="siteAddForm" :rules="addRule" label-position="top">
						<el-form-item label='<%=rb.getString("ZhanZhiMingCheng")%>' prop="siteName" required>
							<el-input v-model="siteAddForm.siteName" style="width: 100%;" maxlength="50"></el-input>
						</el-form-item>
						<el-form-item label='<%=rb.getString("WeiZhi")%>' prop="latitude" required>
							<div style="position: absolute;right: 1px;top: -25px;">
								<span @click="addPickOnMap" style="color: #689CF3;cursor: pointer;display: flex;align-items: center;">
									<i class="el-icon el-icon-draw"></i> 
									<span style="text-decoration: underline;margin-left: 5px;">Pick on MAP</span>
								</span>
							</div>
							<el-input v-model="siteAddForm.latitude" style="width: 100%;">
								<template slot="prepend">
									<div style="width: 60px;"><%=rb.getString("WeiDu")%></div>
								</template>
							</el-input>
						</el-form-item>
						<el-form-item prop="longitude" class="no-padding-top">
							<el-input v-model="siteAddForm.longitude" style="width: 100%;margin-top: 20px;">
								<template slot="prepend">
									<div style="width: 60px;"><%=rb.getString("JingDu")%></div>
								</template>
							</el-input>
						</el-form-item>
						<el-form-item label='<%=rb.getString("TianJiaLeiXing")%>'>
							<el-checkbox-group v-model="siteAddDeviceType" @change="siteAddDeviceTypeChange" size="small">
								<el-checkbox v-if="isGSMEnable" label="GSM" border></el-checkbox>
								<el-checkbox label="eNB" border></el-checkbox>
								<el-checkbox label="gNB" border></el-checkbox>
							</el-checkbox-group>
						</el-form-item>
						<el-form-item label='<%=rb.getString("SheBeiLieBiaoBiaoTi")%>'>
							<el-ctable ref="addTb" style="height: 260px;"
								row-key="serial_number"
								:url="addDeviceURL"
								:query-params="addQuery"
								:show-pager="false"
								@selection-change="addSelectionChange"
							>
								<template slot="toolbar">
									<el-query @query="addQueryClick" style="zoom: 0.8;" class="width-fitcontent" type="normal" placeholder='<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>'></el-query>
								</template>
								<el-table-column type="selection" :reserve-selection="true"></el-table-column>
								<el-table-column width="35">
									<template slot-scope="scope">
										<div style="zoom: 0.98;" :class="{
											'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
											'':scope.row.have_connected==2,
											'conn_exc':scope.row.connection_status=='Exception',
											'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
									</template>
								</el-table-column>
								<el-table-column label='<%=rb.getString("XiaoZhanBianMa") %>' prop="serial_number"></el-table-column>
								<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name"></el-table-column>
								<el-table-column label='<%=rb.getString("Type")%>' prop="stationType" width="50"></el-table-column>
							</el-ctable>
							<el-checkbox v-model="siteAddForm.locationConsistent"
								:true-label="true"
								:false-label="false"
							>
								Device and site locations are synchronized
							</el-checkbox>
						</el-form-item>
					</el-form>
				</div>

				<!-- Info Site -->
				<div v-show="siteType == 'view'" style="padding: 15px;flex: auto;overflow: auto;">
					<el-ctable ref="viewTb" size="small"
						row-key="serialNumber"
						:url="viewDeviceURL"
						:query-params="viewQuery"
						:show-pager="false"
					>
						<template slot="toolbar">
							<el-query @query="viewQueryClick" style="zoom: 0.8;" class="width-fitcontent" type="normal" placeholder='<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>'></el-query>
						</template>
						<el-table-column width="40">
							<template slot-scope="scope">
								<span @click="siteOptClick(scope.row, event)" v-clickoutside="handerClose" class="el-icon el-icon-operation-more"></span>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("XiaoZhanBianMa")%>' prop="serial_number"></el-table-column>
						<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name"></el-table-column>
						<el-table-column label='<%=rb.getString("Type")%>' prop="stationType" width="50"></el-table-column>
					</el-ctable>

					<el-cmenu ref="siteMenu" :data="siteMenus" @click="menuClick"></el-cmenu>
				</div>

				<div v-if="siteType == 'add'" style="padding: 15px;">
					<el-button type="primary" @click="addSiteSubmit"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="siteInfoShow = false"><%=rb.getString("QuXiao")%></el-button>
				</div>
			</div>

			<!-- Details -->
			<div v-show="deviceDetailShow" class="detail-panel">
				<div class="detail-header">
					<span style="font-weight: bold;color: #7A7992;">
						<%=rb.getString("XinXi")%>
						<span style="color: rgba(0,0,0,.16)"> ({{detailForm.code}}) </span>
					</span>
					<i @click="deviceDetailShow = false" class="el-icon el-icon-close"></i>
				</div>
				<div class="detail-body">
					<div class="info-item-cls" label="<%=rb.getString("HostName")%>">{{detailForm.name}}</div>
					<div class="info-item-cls" label="<%=rb.getString("IPDiZhi")%>">{{detailForm.ip}}</div>
					<div class="info-item-cls" label="<%=rb.getString("SheBeiZu")%>">{{detailForm.groupName}}</div>
					<div class="info-item-cls" label="<%=rb.getString("ShiFouJiHuo")%>" v-html="activeFmt(detailForm.active, detailForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("GPSWeiZhi")%>">
						{{detailForm.lat}}, {{detailForm.lon}} 
						<span class="el-icon el-icon-operation-edit" v-show="!sasEnable" style="margin-left: 5px;" @click="toInfoModifyGPS"></span>
					</div>
					<div class="info-item-cls" label="<%=rb.getString("GaoJingJiBie")%>" v-html="alarmFmt(detailForm.alarm, detailForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("MMEZhuangTai")%>" style="position: relative;">
						<div v-if="['','NULL','null',null,undefined].includes(infoForm.NEW_MME_STATUS)">
							<span v-html="oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'left')" ></span>
                            <el-popover title="All MME" popper-class="mmePopoverClass" v-if="oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'list').length > 0" :append-to-body="false">
                                <span style="color:#4d84ff;cursor:pointer;" slot="reference" v-if="oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'list').length > 0">
                                    [ <span v-html="oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'right')"></span> ]
                                </span>
                                <div class="mme-list">
                                    <i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
                                    <div class="mme-list-item" v-for="item in oldMmeStatusFmt(infoForm, infoForm.mmeStatus, 'list')">
                                        <span v-if="item.status == '1'" style="margin-top:10px;" class="el-icon el-icon-status-MME greenIcon"></span>
                                        <span v-if="item.status == '0'" style="margin-top:10px;" class="el-icon el-icon-status-MME redIcon"></span> 
                                        <div class="mme-info">
                                            <span>MME IP : {{item.mmeIp}}</span>
                                            <span v-if="item.status==='0'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEWeiLianJie")%></span>
                                            <span v-if="item.status=='1'"><%=rb.getString("MMEZhuangTai")%> : <%= rb.getString("MMEYiLianJie")%></span>
                                            <span v-if="item.status=='2'"><%=rb.getString("MMEZhuangTai")%> : --</span>
                                            <span v-if="item.status==='' || item.status===null"><%=rb.getString("MMEZhuangTai")%> : </span>
                                        </div>
                                    </div>
                                </div>
                            </el-popover>
						</div>
						<div v-if="!['','NULL','null',null,undefined].includes(infoForm.NEW_MME_STATUS)" style="display: flex;align-items: center;">
							<span v-html="mmesStatusFmt(infoForm.NEW_MME_STATUS)" ></span>
							<el-popover title="All MME" popper-class="mmePopoverClass" :append-to-body="false">
                                <span style="color:#4d84ff;cursor:pointer;" slot="reference">
                                    [ {{parseMMEOnNum(infoForm.NEW_MME_STATUS)}}/{{parseMME(infoForm.NEW_MME_STATUS).length}} ]
                                </span>
                                <div class="mme-list">
                                    <i class="el-icon el-icon-close" style="position: absolute; top: -30px; right: 0px;" onclick="document.body.click()"></i>
                                    <div class="mme-list-item" v-for="item in parseMME(infoForm.NEW_MME_STATUS)">
                                        <span v-if="item.status == '1'" class="el-icon el-icon-status-MME greenIcon"></span>
                                        <span v-if="item.status == '0'" class="el-icon el-icon-status-MME redIcon"></span> 
                                        <div class="mme-info">
                                            <span>MME IP : {{item.mmeIp}}</span>
                                            <span v-if="item.status==='0'"><%=rb.getString("MMEZhuangTai")%> : Disconnected</span>
                                            <span v-if="item.status=='1'"><%=rb.getString("MMEZhuangTai")%> : Connected</span>
                                            <span v-if="item.status=='2'"><%=rb.getString("MMEZhuangTai")%> : --</span>
                                            <span v-if="item.status==='' || item.status===null"><%=rb.getString("MMEZhuangTai")%> : </span>
                                            <span><%=rb.getString("PLMN")%> : {{item.plmnId}}</span>
                                        </div>
                                    </div>
                                </div>
                            </el-popover>
						</div>
					</div>
					<div class="info-item-cls" label="<%=rb.getString("RSMME")%>" v-html="mmeIPfmt(detailForm.mmeStatus, detailForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("SASKaiGuan")%>" v-show="hasSAS" v-html="sasEnableFmt(detailForm.mmeStatus, detailForm)"></div>
					<div class="info-item-cls" label="<%=rb.getString("SASZhuangTai")%>" v-show="hasSAS" v-html="sasStateFmt(detailForm.sasState, detailForm)"></div>
				</div>
			</div>

			<!-- topo list -->
			<div v-show="topolistShow" class="detail-panel" style="width: 780px;">
				<div class="detail-header">
					<span style="font-weight: bold;color: #7A7992;">
						TOPO
						<span style="color: rgba(0,0,0,.16)"> ({{queryParamsEURU.smallCellCode}}) </span>
					</span>
					<div>
						<i v-show="listOrTopoType == 'list' && ['gNB'].includes(queryParamsEURU.type)" @click="changeListOrTopo('topo')" class="el-icon el-icon-operation-topo"></i>
						<i v-show="listOrTopoType == 'topo' && ['gNB'].includes(queryParamsEURU.type)" @click="changeListOrTopo('list')" class="el-icon el-icon-table"></i>
						<i @click="topolistShow = false" class="el-icon el-icon-close" style="margin-left: 15px;"></i>
					</div>
				</div>
				<div class="detail-body" style="padding: 15px 20px;">
					<!-- list -->
					<div v-show="listOrTopoType == 'list'">
						<p style="display: inline-block;">
							<span class="title-icon" style="vertical-align: sub;"></span> 
							<span style="font-size: 14px; font-weight: bold;">HUB</span>
						</p>
						<el-ctable height="200px" class="border" style="border-radius: 5px;"
							:url="euUrl"
							:query-params="queryParamsEURU"
							:rownumber="false"
						>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("LuYouSuoYin") %>" key="route_index" prop="route_index" width="100"></el-table-column>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("ZhuangTai") %>" key="status" prop="status">
								<template slot-scope="scope">
									<div v-if="scope.row.status == '1'">
										<span class='el-icon el-icon-status-conn-on' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ZhengChang") %></span>
									</div>
									<div v-if="scope.row.status == '2'">
										<span class='el-icon el-icon-status-conn-off' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("LiXian") %></span>
									</div>
									<div v-if="scope.row.status == '3'">
										<span class='el-icon el-icon-status-alarm' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("GaoJingGuanLi") %></span>
									</div>
									<div v-if="scope.row.status == '4'">
										<span class='el-icon el-icon-status-upgrading' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ShengJiZhong") %></span>
									</div>
									<div v-if="scope.row.status == '5'">
										<span class='el-icon el-icon-status-upgrade-success' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ShengJi") %><%=rb.getString("ChengGong") %></span>
									</div>
								</template>
							</el-table-column>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("XiaoZhanBianMa") %>" key="serial_number" prop="serial_number" width="180"></el-table-column>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("MoKuaiXingHao") %>" key="model_name" prop="model_name"></el-table-column>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("RuanJianBanBen") %>" key="software_version" prop="software_version" width="140"></el-table-column>

							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("LuYouSuoYin") %>" key="RouteIndex" prop="RouteIndex" width="100"></el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("ZhuangTai") %>" key="Status" prop="Status">
								<template slot-scope="scope">
									<div v-if="scope.row.Status == '1'">
										<span class='el-icon el-icon-status-conn-on euStatus'></span><span class="statusTip"><%=rb.getString("ZhengChang") %></span>
									</div>
									<div v-if="scope.row.Status == '2'">
										<span class='el-icon el-icon-status-conn-off euStatus'></span><span class="statusTip"><%=rb.getString("LiXian") %></span>
									</div>
									<div v-if="scope.row.Status == '3'">
										<span class='el-icon el-icon-status-alarm euStatus'></span><span class="statusTip"><%=rb.getString("GaoJingGuanLi") %></span>
									</div>							
								</template>
							</el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("XiaoZhanBianMa") %>" key="SerialNumber" prop="SerialNumber"></el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("MoKuaiXingHao") %>" key="ModelName" prop="ModelName"></el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("RuanJianBanBen") %>" key="SoftwareVersion" prop="SoftwareVersion"></el-table-column>
						</el-ctable>

						<p style="display: inline-block; margin-top: 20px;">
							<span class="title-icon" style="vertical-align: sub;"></span> 
							<span style="font-size: 14px; font-weight: bold;">RRU</span>
						</p>
						<el-ctable height="200px" class="border" style="border-radius: 5px;"
							:url="ruUrl"
							:query-params="queryParamsEURU"
							:rownumber="false"
						>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label='<%=rb.getString("LuYouSuoYin") %>' key="enb_route_index" prop="route_index" width="100"></el-table-column>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label='<%=rb.getString("ZhuangTai") %>' key="enb_status" prop="status">
								<template slot-scope="scope">
									<div v-if="scope.row.status == '1'">
										<span class='el-icon el-icon-status-conn-on' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ZhengChang") %></span>
									</div>
									<div v-if="scope.row.status == '2'">
										<span class='el-icon el-icon-status-conn-off' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("LiXian") %></span>
									</div>
									<div v-if="scope.row.status == '3'">
										<span class='el-icon el-icon-status-alarm' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("GaoJingGuanLi") %></span>
									</div>
									<div v-if="scope.row.status == '4'">
										<span class='el-icon el-icon-status-upgrading' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ShengJiZhong") %></span>
									</div>
									<div v-if="scope.row.status == '5'">
										<span class='el-icon el-icon-status-upgrade-success' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("ShengJi") %><%=rb.getString("ChengGong") %></span>
									</div>
								</template>
							</el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label='<%=rb.getString("LuYouSuoYin") %>' key="gnb_RouteIndex" prop="RouteIndex" width="100"></el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label='<%=rb.getString("ZhuangTai") %>' key="nb_Status" prop="Status">
								<template slot-scope="scope">
									<div v-if="scope.row.Status == '1'">
										<span class='el-icon el-icon-status-conn-on euStatus'></span><span class="statusTip"><%=rb.getString("ZhengChang") %></span>
									</div>
									<div v-if="scope.row.Status == '2'">
										<span class='el-icon el-icon-status-conn-off euStatus'></span><span class="statusTip"><%=rb.getString("LiXian") %></span>
									</div>
									<div v-if="scope.row.Status == '3'">
										<span class='el-icon el-icon-status-alarm euStatus'></span><span class="statusTip"><%=rb.getString("GaoJingGuanLi") %></span>
									</div>							
								</template>
							</el-table-column>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label='<%=rb.getString("XiaoZhanBianMa") %>' key="enb_serial_number" width="120" prop="serial_number"></el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label='<%=rb.getString("XiaoZhanBianMa") %>' key="gnb_SerialNumber" width="120" prop="SerialNumber"></el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label="TxGain" prop="TxGain"></el-table-column>
							<el-table-column label='<%=rb.getString("FangSheZhuangTai") %>' width="100" prop="RFTxStatus">
								<template slot-scope="scope">
									<div v-if="['false','0'].includes(scope.row.rf_tx_status) || ['false','0'].includes(scope.row.RFTxStatus)">
										<span class='el-icon el-icon-status-disable' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("GuanBi") %></span>
									</div>
									<div v-if="['true','1'].includes(scope.row.rf_tx_status) || ['true','1'].includes(scope.row.RFTxStatus)">
										<span class='el-icon el-icon-status-enable' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("KaiQi") %></span>
									</div>
								</template>
							</el-table-column>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("MoKuaiXingHao") %>" key="enb_model_name" prop="model_name"></el-table-column>
							<el-table-column v-if="['eNB'].includes(queryParamsEURU.type)" label="<%=rb.getString("RuanJianBanBen") %>" key="enb_software_version" prop="software_version" width="140"></el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label='<%=rb.getString("MoKuaiXingHao") %>' width="100" key="gnb_ModelName" prop="ModelName"></el-table-column>
							<el-table-column v-if="['gNB'].includes(queryParamsEURU.type)" label='<%=rb.getString("RuanJianBanBen") %>' width="130" key="gnb_SoftwareVersion" prop="SoftwareVersion"></el-table-column>
						</el-ctable>
					</div>
					
					<div v-show="listOrTopoType == 'list' && ['GSM'].includes(queryParamsEURU.type)">
						<p style="display: inline-block;">
							<span class="title-icon" style="vertical-align: sub;"></span> 
							<span style="font-size: 14px; font-weight: bold;">BTS</span>
						</p>
						<el-ctable height="400px" class="border" style="border-radius: 5px;"
							:url="btsUrl"
							:query-params="queryParamsEURU"
							:rownumber="true"
						>
							<el-table-column label="IpaUnitld" prop="IpaUnitld"></el-table-column>
							<el-table-column label="IpOmIRemoteIp" prop="OmIRemoteIp"></el-table-column>
							<el-table-column label="IpOmIRemoteIpBak" prop="OmIRemoteIpBak"></el-table-column>
							<el-table-column label="Status" prop="BscLinkStatus">
								<template slot-scope="scope">
									<div v-if="scope.row.product == 'BTS' || true">
										<span v-if="scope.row.BscLinkStatus == '0'"><%=rb.getString("MMEWeiLianJie")%></span>
										<span v-if="scope.row.BscLinkStatus == '1'"><%=rb.getString("MMEYiLianJie")%></span>
									</div>
									<div v-if="scope.row.product == 'BSC'">--</div>
								</template>
							</el-table-column>
						</el-ctable>
					</div>
					<!-- topo -->
					<div v-show="listOrTopoType == 'topo'">
						<div id="device_vis_topo" style="height: 450px;"></div>
					</div>
				</div>
			</div>
		</div>

		<!-- Modify Site -->
		<el-dialog ref="siteModify" :class="{'pick-model': siteModifyHidden}" style="top: 100px;"
			:modal="false"
			:close-on-click-modal="false"
			:append-to-body="false"
			title='<%=rb.getString("DingWei")%>'
			:width="400"
			:visible.sync="siteModifyShow"
		>
			<el-form ref="modifyForm" :model="siteModifyForm" :rules="modifyRule" label-position="top" size="mini">
				<div v-if="siteModifyForm.siteName" style="font-weight: bold;padding-bottom: 20px;">
					<span><%=rb.getString("ZhanZhiMingCheng")%>: </span>
					{{siteModifyForm.siteName}}
				</div>
				<div v-if="siteModifyForm.serialNumber" style="font-weight: bold;padding-bottom: 20px;">
					<span><%=rb.getString("XiaoZhanBianMa")%>: </span>
					{{siteModifyForm.serialNumber}}
				</div>
				</el-form-item>
				<el-form-item label='<%=rb.getString("WeiDu")%>' style="margin-bottom: 20px;" prop="latitude">
					<el-input v-model="siteModifyForm.latitude"></el-input>
				</el-form-item>
				<el-form-item label='<%=rb.getString("JingDu")%>' prop="longitude">
					<el-input v-model="siteModifyForm.longitude"></el-input>
				</el-form-item>
			</el-form>

			<div slot="footer" style="padding: 0 10px;display: flex;justify-content: space-between;align-items: center;">
				<span @click="modifyPickOnMAP" style="color: #689CF3;cursor: pointer;display: flex;align-items: center;">
					<i class="el-icon el-icon-draw"></i> 
					<span style="text-decoration: underline;margin-left: 5px;">Pick on MAP</span>
				</span>
				<span>
					<el-button type="primary" @click="modifySiteSubmit"><%=rb.getString("QueDing")%></el-button>
					<el-button @click="siteModifyShow = false"><%=rb.getString("QuXiao")%></el-button>
				</span>
			</div>
		</el-dialog>

		<!-- Add Device to Site -->
		<el-dialog ref="siteDevice" style="top: 50px;"
			:modal="false"
			:close-on-click-modal="false"
			:append-to-body="false"
			title='<%=rb.getString("TianJia")%>'
			:width="800"
			:visible.sync="addSiteDeviceShow"
		>
			<el-form ref="siteDeviceFm" :model="deviceToSiteForm" size="small">
				<el-form-item label='<%=rb.getString("TianJiaLeiXing")%>'>
					<el-checkbox-group v-model="deviceToSiteType" @change="deviceToSiteTypeChange" size="small">
						<el-checkbox v-if="isGSMEnable" label="GSM" border></el-checkbox>
						<el-checkbox label="eNB" border></el-checkbox>
						<el-checkbox label="gNB" border></el-checkbox>
					</el-checkbox-group>
				</el-form-item>
				<el-form-item>
					<el-ctable ref="deviceToSiteTb" class="border" style="height: 300px;border-radius: 5px;"
						row-key="serial_number"
						:url="addDeviceURL"
						:query-params="deviceToSiteQuery"
						:show-pager="false"
						@selection-change="deviceToSiteSelectChange"
					>
						<template slot="toolbar">
							<div style="display: flex;align-items: center;">
								<span style="font-weight: bold;font-size: 14px;"><%=rb.getString("KPISheBei")%></span>
								<el-query @query="deviceToSiteQueryClick" type="normal" placeholder="SN/Cell Name"></el-query>
							</div>
						</template>
						<el-table-column type="selection" :reserve-selection="true"></el-table-column>
						<el-table-column width="35">
							<template slot-scope="scope">
								<div style="zoom: 0.98;" :class="{
									'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
									'':scope.row.have_connected==2,
									'conn_exc':scope.row.connection_status=='Exception',
									'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
							</template>
						</el-table-column>
						<el-table-column label='<%=rb.getString("XiaoZhanBianMa") %>' prop="serial_number"></el-table-column>
						<el-table-column label='<%=rb.getString("HostName")%>' prop="host_name"></el-table-column>
						<el-table-column label='<%=rb.getString("Type")%>' prop="stationType" width="50"></el-table-column>
					</el-ctable>
					
					<el-checkbox style="margin-top: 10px;"
						v-model="deviceToSiteForm.locationConsistent"
						:true-label="true"
						:false-label="false"
					>
						After checking, the longitude and latitude of the base station will be consistent with the site.
					</el-checkbox>
				</el-form-item>
			</el-form>
			
			<div slot="footer" style="padding-left: 10px;">
				<el-button @click="addSiteDeviceSubmit" type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="addSiteDeviceShow = false"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-dialog>

		<el-dialog ref="gpsdg" title="<%=rb.getString("WeiZhi")%>" :visible.sync="gpsdgVisible" :width="400" :append-to-body="false">
			<el-form ref="gpsform" :model="gpsForm" :rules="gpsRules" label-position="top">
				<el-form-item v-if="activeName == 'ENB'">
					SN<%=rb.getString("MaoHao")%> {{curSerialNumber}}
				</el-form-item>
				<el-form-item v-else>
					<%=rb.getString("MACDiZhi")%><%=rb.getString("MaoHao")%> {{curMacAddress}}
				</el-form-item>
				
				<el-form-item label="<%=rb.getString("WeiDu")%><%=rb.getString("MaoHao")%>" prop="lat">
					<el-input v-model="gpsForm.lat"></el-input>
				</el-form-item>
				<el-form-item label="<%=rb.getString("JingDu")%><%=rb.getString("MaoHao")%>" prop="lon">
					<el-input v-model="gpsForm.lon"></el-input>
				</el-form-item>
				<el-form-item label="<%=rb.getString("GaoDu")%><%=rb.getString("MaoHao")%>" prop="height">
					<el-input v-model="gpsForm.height"></el-input>
				</el-form-item>
				
				<el-form-item v-if="gpsDeviceType == 'enb'" prop="mechanical_downtilt" label="<%=rb.getString("JiXieXiaQingJiao")%>">
					<el-input v-model="gpsForm.mechanical_downtilt" maxLength="5" placeholder="Range: 0-9"><el-input>
				</el-form-item>
				<el-form-item v-if="gpsDeviceType == 'enb'" prop="vertical_3dB_beam_width" label="<%=rb.getString("ChuiZhiBoSuKuanDu")%>">
					<el-input v-model="gpsForm.vertical_3dB_beam_width" maxLength="5" placeholder="Range: 1-9"><el-input>
				</el-form-item>
				<el-form-item v-if="gpsDeviceType == 'enb'" prop="horizontal_azimuth" label="<%=rb.getString("ShuiPinFangWeiJiao")%>">
					<el-input v-model="gpsForm.horizontal_azimuth" maxLength="5" placeholder="Range: 0-359"><el-input>
				</el-form-item>
			</el-form>
			<div>
				<el-button type="primary" @click="setGPS"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="gpsdgVisible = false"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-dialog>

		<el-dialog ref="gpsmove" :width="400" 
			:append-to-body="false"
			:visible.sync='gpsmoveVisible' 
			title='<%=rb.getString("QueRen")%>'
			@close="cancelMove">
			<div>
				<%=rb.getString("ChongSheGPS")%><br> 
				<%=rb.getString("XinJingWeiDu")%>: {{gpsmvoeForm.lat}}, {{gpsmvoeForm.lon}}
			</div>
			<div style="padding-top: 10px;">
				<el-button type="primary" @click="confirmMove"><%=rb.getString("QueDing")%></el-button>
				<el-button @click="cancelMove"><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-dialog>
	</div>
<div>

<script type="text/javascript">
	var topoOverViewEnable = scenarioKey == 'S00013';
	//RSRP
	var lowVal = localStorage.getItem("rsrp1"),
		highVal = localStorage.getItem("rsrp2");
	
	var isMeasuring = false,
		measureObj = {
			drawing: false,
			layer: null,
			clickHandler: null,
			measure: function() {
				isMeasuring = true;
				var map = globMap;

				if(topovm.activeType=='site') {
					map = globSiteMap;
				}

				// 避免多次drawing
				if (this.drawing) {
					return
				}

				if(this.layer) {
					map.removeLayer(this.layer);
					document.querySelector('#map').click();
				}

				// interactive = false 避免用户双击map无效
				const layer = L.polyline([], {
					interactive: false
				}).addTo(map);
				this.layer = layer;

				// 绘制mousemove line
				const tempLayer = L.polyline([], {
					interactive: false
				}).addTo(map);
				let tempPoints = [];

				// 结束绘制
				const remove = () => {
					map.removeLayer(layer);
					map.removeLayer(popup);
					dblclickHandler();
					isMeasuring = false;
					$('.flex-bt-cls').removeClass('selected');
				};

				// popup 展示距离
				const popup = L.popup({
					autoClose: false,
					closeButton: false
				});

				// 自定义 展示框
				const setTipText = content => {
					const el = document.createElement("div");
					el.className = "measure-marker line-text";

					const text = document.createElement("span");
					text.className = "measure-text white-space-nowrap";
					text.innerHTML = content;

					const close = document.createElement("span");
					close.className = "measure-close el-icon-close";

					close.addEventListener("click", () => {
						remove();
					});

					el.appendChild(text);
					el.appendChild(close);

					return el;
				};

				const clickHandler = e => {
					layer.addLatLng(e.latlng);
					tempPoints[0] = e.latlng;
					this.drawing = true;
					map.doubleClickZoom.disable();

					const len = turf.length(layer.toGeoJSON(), { units: "kilometers" });

					popup
					.setLatLng(e.latlng)
					.setContent(setTipText(len.toFixed(2) + " km"))
					.openOn(map);
				};

				this.clickHandler = clickHandler;

				const mousemoveHandler = e => {
					if (tempPoints.length) {
					tempPoints[1] = e.latlng;
					tempLayer.setLatLngs(tempPoints);
					}
				};

				// 双击结束， 移除事件是良好的编程习惯
				const dblclickHandler = e => {
					tempPoints = null;
					map.removeLayer(tempLayer);
					//tempLayer.remove();
					this.drawing = false
					map.doubleClickZoom.enable();

					map.off("click", clickHandler, true);
					map.off("mousemove", mousemoveHandler);
					map.off("dblclick", dblclickHandler);
					this.clickHandler = null;

					$('.flex-bt-cls').removeClass('selected');
				};

				map.on("click", clickHandler, true);
				map.on("mousemove", mousemoveHandler);
				map.on("dblclick", dblclickHandler);
				this.drawing = true;
			}
		};

	function hideRepeatedList() {
		$('#topo_container .repeated-list').fadeOut();
	}
	$('body').off('click',hideRepeatedList).on('click',hideRepeatedList);
	
	var globMap, globSiteMap, groupName = '';
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
					value = value + '';

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
						if(rule.required == true) {
							cb('Required');
						}else {
							cb();
						}
					}
				},
				/**
				* 经度校验
				* @param rule{object}：校验配置的规则
				* @param value{string}: 经度值
				* @param cb{function}：回调方法
				**/
				lonValid = function(rule,value,cb){
					value = value + '';

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
						if(rule.required == true) {
							cb('Required');
						}else {
							cb();
						}
					}
				},// 高度 表单验证规则
				heightValidator = function(rule,value,cb) {
					if(value) {
						if(value<=99999999 && value-0>=0) {
							cb();
						}else {
							cb('<%=rb.getString("QuZhiFanWei")%>: 0-99999999');
						}
					}else {
						cb();
					}
				},
				validSiteName = (rule,value,cb) => {
					var trimVal = value.trim();

					if(trimVal === '') {
						cb('Required');
					}else {
						cb();
					}
				},
				validMechanicalDowntilt = function(rule,value,callback) {
					var msg = 'Range: 0-9';
					
					if(value === '') {
						callback();
					}else {
						if(isNaN(value) || value - 0 < 0 || value - 9 > 0) {
							callback(msg)
						}else {
							callback();
						}
					}
				},
				validVertical3dB = function(rule,value,callback) {
					var msg = 'Range: 1-9';
					
					if(value === '') {
						callback();
					}else {
						if(isNaN(value) || value - 1 < 0 || value - 9 > 0) {
							callback(msg)
						}else {
							callback();
						}
					}
				},
				validHorizontalAzim = function(rule,value,callback) {
					var msg = 'Range: 0-359';
					
					if(value === '') {
						callback();
					}else {
						if(isNaN(value) || value - 0 < 0 || value - 359 > 0) {
							callback(msg)
						}else {
							callback();
						}
					}
				};
			
			return {
				topoOverViewEnable: topoOverViewEnable,
				gpsDeviceType: '',
				activeType: 'group',
				selectedCode: '',

				groupIds: '',
				limit: 100, // 过滤数据: 阀值
				links: {}, // enb为key，cpe存于数组中：enbCode:[cpeCode1, cpeCode2, ...]，建立 1:n 关系
				activeName: 'ENB',
				groupURL: '${ctx}/system/deviceGroup/getDeviceGroupList.action',
				deviceURL: '',
				cpeDeviceURL: '',
				// 设备组查询参数
				groupQueryParams: {
					search_text: ''
				},
				// enb列表查询参数
				enbQueryParams: {
					device_group_id: '',
					device_type: 'ENB',
					search_text: '',
					operator_code: operator_code,
					no_gps: ''
				},
				// cpe列表查询参数
				cpeQueryParams: {
					device_group_id: '',
					device_type: 'CPE',
					search_text: '',
					operator_code: operator_code,
					no_gps: ''
				},
				// 显示状态表单 -- 提交用
				form: {
					deviceType: ['enb','cpe'],
					deviceStatus: ['on','off'],
					activeStatus: [],
					sasStatus: []
				},
				// 显示状态表单 -- 界面展示用
				statusForm: {
					deviceType: ['enb','cpe'],
					deviceStatus: ['on','off'],
					activeStatus: [],
					sasStatus: []
				},
				statusType: 'deviceStatus',
				// gps设置表单
				gpsForm: {
					lat: '', // 纬度
					lon: '',
					height: '',
					mechanical_downtilt: '',
					vertical_3dB_beam_width: '',
					horizontal_azimuth: '',
				},
				gpsmvoeForm: {
					lat: '', // 纬度
					lon: '', // 经度
					olatlng: '',
					cell_code: '',
					operator_code: operator_code,
					target: '',
				},
				// gps域校验规则
				gpsRules: {
					lat:[
						{validator: latValid}
					],
					lon:[
						{validator: lonValid}
					],
					height: [{validator: heightValidator}],
					mechanical_downtilt: [{validator: validMechanicalDowntilt}],
					vertical_3dB_beam_width: [{validator: validVertical3dB}],
					horizontal_azimuth: [{validator: validHorizontalAzim}]
				},
				infoForm: {
					cellCode: '',
					active: '',
					alarm: {},
					code: '',
					groupName: '',
					ip: '',
					lat: '',
					lon: '',
					mmeEnable: '',
					mmePool1: '',
					mmePool2: '',
					mmeStatus: '',
					name: '',
					online: '',
					serverList: '',
					sasEnable: '',
					sasState: '',
					sn: '',
					NEW_MME_STATUS: '',
					RSRP0: '',
					RSRP1: '',
					pci: '',
					mechanical_downtilt: '',
					vertical_3dB_beam_width: '',
					horizontal_azimuth: '',
					minRadius: '',
					radius: ''
				},
				cpeNodes: [], // 记录获取的cpe节点
				enbNodes: [], // 记录获取的enb节点
				prevNodes: [], // 缓存的上次绘制节点
				gpsdgVisible: false,
				gpsmoveVisible: false,
				allNodes: [], // cpe、enb节点合集
				curDeviceCode: '',
				curSerialNumber: '',
				curMacAddress: '',
				sasEnable: true,
				groupShow: true,
				curRow: {},
				groupNames: '',
				firtLoad: true,
				repeatedMap: {},
				repeatedHideNodes: [],
				infoShow: false,
				repeatData: [],
				repeatShow: false,
				repeatedQueryText: '',

				siteInfoShow: false,
				siteMenus: [],
				siteListURL: '',
				siteQuery: {
					siteName: '',
					rd: ''
				},
				siteType: '',
				currentSiteRow: {},
				addDeviceURL: '${ctx}/site/getStationList.action',
				viewDeviceURL: '${ctx}/cell/topo/getDeviceInfoList.action',
				siteAddDeviceType: ['eNB'],
				deviceToSiteType: ['eNB'],
				addQuery: {
					deviceType: 'eNB',
					searchText: '',
					page: 1,
					rows: 20
				},
				deviceToSiteQuery: {
					deviceType: 'eNB',
					searchText: '',
					page: 1,
					rows: 20
				},
				viewQuery: {
					queryType: 'groupid',
					device_type: 'ENB',
					operator_code: operator_code,
					siteName: '',
					search_text: '',
					rd: ''
				},
				siteAddForm: {
					siteName: '',
					latitude: '',
					longitude: '',
					sns: '',
					locationConsistent: false,
				},
				deviceToSiteForm: {
					siteName: '',
					sns: '',
					locationConsistent: false,
				},
				addRule: {
					siteName: [{validator: validSiteName}],
					latitude: [{validator: latValid, required: true}],
					longitude: [{validator: lonValid, required: true}]
				},
				siteModifyForm: {
					serialNumber: '',
					cellCode: '',
					siteName: '',
					latitude: '',
					longitude: '',
				},
				modifyRule: {
					latitude: [{validator: latValid, required: true}],
					longitude: [{validator: lonValid, required: true}],
				},
				siteModifyShow: false,
				siteModifyHidden: false,
				deviceDetailShow: false,
				siteAddHidden: false,
				detailForm: {
					cellCode: '',
					active: '',
					alarm: {},
					code: '',
					groupName: '',
					ip: '',
					lat: '',
					lon: '',
					mmeEnable: '',
					mmePool1: '',
					mmePool2: '',
					mmeStatus: '',
					name: '',
					online: '',
					serverList: '',
					sasEnable: '',
					sasState: '',
					sn: '',
					NEW_MME_STATUS: '',
					RSRP0: '',
					RSRP1: ''
				},
				addSiteDeviceShow: false,
				topolistShow: false,
				allSiteLatlonNodes: [],
				listOrTopoType: 'list',
				dataURL: '',
				euUrl: '',
				ruUrl: '',
				btsUrl: '',
				queryParamsEURU:{
					smallCellCode: '',
					type: ''
				},
				
				siteEnable: supportTopoSite,
				siteDeviceType: ['enb'],
				siteCheckAll: false,
				isIndeterminate: true,
				isGSMEnable: supportGSM == true,

				overViewList: []
			};
		},
		computed: {
			cpeEnable() {
				return writableMap['CODE_CPE_MONITOR'] != undefined;
			},
			hasSAS() {
				return writableMap['CODE_ADVANCE_SAS'] != undefined;
			},
			isLocal() {
				return isLocal == 'true';
			},
			repeatedTbData() {
				var vm = this;

				return vm.repeatData.filter(function(row) {
					var code = row.code||'',
						name = row.name || '';

					return code.indexOf(vm.repeatedQueryText) >= 0 || name.indexOf(vm.repeatedQueryText) >= 0;
				});
			},
			infoPanelCls() {
				return {
					'info-panel': true,
					'show': this.infoShow || this.repeatShow
				};
			},
			sitePanelCls() {
				return {
					'site-panel': true,
					'show': this.siteInfoShow,
					'pick-model': this.siteAddHidden
				};
			},
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
			deviceInfos() {
				var vm = this,
					allNodes = vm.allNodes.concat(vm.repeatedHideNodes).filter(function(item){
						return !['',null].includes(item.lat) && !['',null].includes(item.lon);
					}),
					
					allEnbNodes = vm.allNodes.concat(vm.repeatedHideNodes).filter(function(item){
						return !['',null].includes(item.lat) && !['',null].includes(item.lon) && item.type == 'enb';
					}),
					allCpeNodes = vm.allNodes.concat(vm.repeatedHideNodes).filter(function(item){
						return !['',null].includes(item.lat) && !['',null].includes(item.lon) && item.type == 'cpe';
					}),
					
					enbOnlineList = allNodes.filter(function(item){
						return item.online == 'on' && item.type == 'enb';
					}),
					cpeOnlineList = allNodes.filter(function(item){
						return item.online == 'on' && item.type == 'cpe';
					}),
					
					enbActiveList = allNodes.filter(function(item){
						return item.active == 'yes' && item.type == 'enb';
					}),
					cpeActiveList = allNodes.filter(function(item){
						return item.active == 'yes' && item.type == 'cpe';
					}),
					
					enbUnregList = allNodes.filter(function(item){
						return item.sasState == 'Unregistered' && item.type == 'enb';
					}),
					enbRegList = allNodes.filter(function(item){
						return item.sasState == 'Registered' && item.type == 'enb';
					}),
					enbGrantedList = allNodes.filter(function(item){
						return item.sasState == 'Granted' && item.type == 'enb';
					}),
					enbAuthList = allNodes.filter(function(item){
						return item.sasState == 'Authorized' && item.type == 'enb';
					}),
					
					
					onlineList = allNodes.filter(function(item){
						return item.online == 'on';
					}),
					activeList = allNodes.filter(function(item){
						return item.active == 'yes';
					}),
					unregList = allNodes.filter(function(item){
						return item.sasState == 'Unregistered';
					}),
					regList = allNodes.filter(function(item){
						return item.sasState == 'Registered';
					}),
					grantedList = allNodes.filter(function(item){
						return item.sasState == 'Granted';
					}),
					authList = allNodes.filter(function(item){
						return item.sasState == 'Authorized';
					});

				return {
					onlineCount: onlineList.length,
					activeCount: activeList.length,
					total: allNodes.length,
					unregistered: unregList.length,
					registered: regList.length,
					granted: grantedList.length,
					authorized: authList.length,
					
					enbTotal: allEnbNodes.length,
					enbOnlineCount: enbOnlineList.length,
					cpeTotal: allCpeNodes.length,
					cpeOnlineCount: cpeOnlineList.length,
					enbActiveCount: enbActiveList.length,
					cpeActiveCount: cpeActiveList.length,

					enbunregistered: enbUnregList.length,
					enbregistered: enbRegList.length,
					enbgranted: enbGrantedList.length,
					enbauthorized: enbAuthList.length
				};
			}
		},
		watch: {
			form: {
				deep: true,
				handler(n) {
					var vm = this;
					
					//vm.filterRender(vm.allNodes, globMap, true);
					vm.getTopoInfos(vm.groupIds);
					vm.hidePanel();
				}
			},
			siteAddDeviceType: function(val) {
				var vm = this;

				vm.addQuery.deviceType = val.join(',');
			},
			deviceToSiteType: function(val) {
				var vm = this;

				vm.deviceToSiteQuery.deviceType = val.join(',');
			},
			siteInfoShow: function(val) {
				var vm = this;

				if(val == false) {
					vm.deviceDetailShow = false;
					vm.topolistShow = false;
				}
			}
		},
		methods: {
			initOverView() {
				var vm = this,
					url = '${ctx}/cell/cpeinfos/getDeviceGroupNames.action?rd=' + Math.random();

				if(vm.topoOverViewEnable !== true) {
					return;
				}

				vm.overViewList = [];
				axios.post(url).then((res)=>{
					var list = res.data || [];

					list.map(function(code){
						vm.overViewList.push({
							area: code,
							btsTotal: '',
							btsOnline: '',
							enbTotal: '',
							enbOnline: '',
							gnbTotal: '',
							gnbOnline: '',
							onlineTotal: '',
							total: ''
						})
					})

					// 获取 BTS Total
					axios.post('${ctx}/cell/cpeinfos/getBtsTotal.action?rd=' + Math.random()).then((res)=>{
						var data = res.data;

						vm.overViewList.map(function(item){
							item.btsTotal = data[item.area];
						})
						vm.staticTotal();
					})
					// 获取 BTS Online
					axios.post('${ctx}/cell/cpeinfos/getBtsOnlineCount.action?rd=' + Math.random()).then((res)=>{
						var data = res.data;

						vm.overViewList.map(function(item){
							item.btsOnline = data[item.area];
						})
						vm.staticTotal();
					})
					// 获取 eNB Total
					axios.post('${ctx}/cell/cpeinfos/getEnbTotal.action?rd=' + Math.random()).then((res)=>{
						var data = res.data;

						vm.overViewList.map(function(item){
							item.enbTotal = data[item.area];
						})
						vm.staticTotal();
					})
					// 获取 eNB Online
					axios.post('${ctx}/cell/cpeinfos/getEnbOnlineCount.action?rd=' + Math.random()).then((res)=>{
						var data = res.data;

						vm.overViewList.map(function(item){
							item.enbOnline = data[item.area];
						})
						vm.staticTotal();
					})
					// 获取 gNB Online
					axios.post('${ctx}/cell/cpeinfos/getGnbTotal.action?rd=' + Math.random()).then((res)=>{
						var data = res.data;

						vm.overViewList.map(function(item){
							item.gnbTotal = data[item.area];
						})
						vm.staticTotal();
					})
					// 获取 gNB Total
					axios.post('${ctx}/cell/cpeinfos/getGnbOnlineCount.action?rd=' + Math.random()).then((res)=>{
						var data = res.data;

						vm.overViewList.map(function(item){
							item.gnbOnline = data[item.area];
						});
						vm.staticTotal();
					})
				})
			},
			staticTotal() {
				var vm = this;

				vm.overViewList.map(function(item){
					item.onlineTotal = ((item.btsOnline||0) - 0) + ((item.enbOnline||0) - 0) + ((item.gnbOnline||0) - 0);
					item.total = ((item.btsTotal||0) - 0) + ((item.enbTotal||0) - 0) + ((item.gnbTotal||0) - 0);
				});
			},
			confirmMove() {
				var vm = this,
					form = vm.gpsmvoeForm,
					olatlng = form.olatlng;

				$.ajax({
					url: '${ctx}/cell/topo/setLocationInfo.action',
					type: 'post',
					dataType: 'json',
					data: {
						longitude: form.lon,
						latitude: form.lat,
						setType: 'topo',
						cell_code: form.cell_code,
						operator_code: operator_code
					},
					success: function(data){
						if(data['success']){
							var latlonKey = olatlng.lat+'-'+olatlng.lng,
								repeatCodes = vm.repeatedMap[latlonKey];
								
							if(repeatCodes && repeatCodes.length>0){
								var topCode = repeatCodes.splice(0,1);

								vm.repeatedHideNodes.map(function(item){
									if(item.code == topCode) {
										vm.allNodes.push(item);
										vm.prevNodes.push(item);
									}
								});

								var idx = -1;
								vm.repeatedHideNodes.map(function(item, i){
									if(item.code == topCode) idx = i;
								});
								// 替换隐藏节点 -- 原top位置节点，替换至隐藏节点队列中
								if(idx>-1) vm.repeatedHideNodes.splice(idx,1);
							}

							refreshTopo({
								cellCode: form.cell_code,
								lat: form.lat,
								lon: form.lon,
								olat: form.lat,
								olon: form.lon
							});
							vm.$refs.enbDevice.refresh();
							if(writableMap.CODE_CPE_MONITOR != undefined) {
								vm.$refs.cpeDevice.refresh();
							}
						}else{
							toast(data['message'],$('#omc_app_ctn'));
						}
						vm.gpsmoveVisible = false;
					}
				})
			},
			cancelMove() {
				var vm = this,
					form = vm.gpsmvoeForm,
					olatlng = form.olatlng,
					target = form.target;
				
				target.setLatLng([olatlng.lat, olatlng.lng]);
				vm.gpsmoveVisible = false;
			},
			queryRepeated(text) {
				var vm = this;

				vm.repeatedQueryText = text;
			},
			measure() {
				measureObj.measure();
			},
			activeFmt(value, row) {
				value = value == 'yes'? 
								  '<i class="el-icon el-icon-status-active online margin-right-5"></i><%=rb.getString("JiHuo")%>'
								  :'<i class="el-icon el-icon-status-active inactive margin-right-5"></i><%=rb.getString("QuJiHuo")%>';

				return value;
			},
			oldMmeStatusFmt(nodeInfo,value,type){
				var vm = this,
					rowData = {
						s1siglinkserverlist: nodeInfo.serverList,
						mmeEnable: nodeInfo.mmeEnable,
						mmeStatus: nodeInfo.mmeStatus,
						mme_pool_1: nodeInfo.mmePool1,
						mme_pool_2: nodeInfo.mmePool2
					},
					mmeStatus,textVal,value,hasDisconn=false,hasConn=false,mmeDataList=[],leftStr='',parseMMEOnNum=0;
			
				if (value == null || value == "") {
					mmeStatus ='';
					textVal = '--'
				}else{
					if (value == "1") {
						mmeDataList.push({mmeIp:rowData.s1siglinkserverlist,status:'1'})
						mmeStatus ='';
						textVal = '<%= rb.getString("MMEYiLianJie")%>'
					} else if (value == "0") {
						mmeDataList.push({mmeIp:rowData.s1siglinkserverlist,status:'0'})
						mmeStatus ='offStatusCls';
						textVal = '<%= rb.getString("MMEWeiLianJie")%>'
					} else if (value == "2") {
						mmeStatus ='';
						textVal = '--'
					} else{
						var mmePools = value.split(",");
						for(var i = 0; i < mmePools.length; i++ ){
							var mme = mmePools[i];
							var mmeArr = mme.split("=");
							
							if(mmeArr.length == 2 && "mme1" == mmeArr[0]){
								mmeDataList.push({mmeIp:rowData.mme_pool_1,status:mmeArr[1]})
							}else if(mmeArr.length == 2 && "mme2" == mmeArr[0]){
								mmeDataList.push({mmeIp:rowData.mme_pool_2,status:mmeArr[1]})
							}
						}
						mmeDataList.map((item)=>{
							if (item.status == '1'){
								hasConn = true;
							}else if(item.status == '0'){
								hasDisconn = true
							}
						})
						if (hasDisconn == true){
							if (hasConn == false) {
								mmeStatus ='offStatusCls';
								textVal = '<%= rb.getString("MMEWeiLianJie")%>'
							}else{
								mmeStatus ='offOrOnStatusCls';
								textVal = '<%= rb.getString("MMEYiLianJie")%>'
							}
						}else {
							mmeStatus ='';
							textVal = '<%= rb.getString("MMEYiLianJie")%>'
						}
					}
				}
				mmeDataList.map((item)=>{
					if (item.status == '1'){
						parseMMEOnNum += 1;
					}
				})
				if(type == 'left'){
					value = "<span class='"+ mmeStatus +"'>" + textVal +"</span>"
				}else if(type == 'right'){
					value = parseMMEOnNum + '/' + mmeDataList.length
				}else if(type == 'list'){
					value = mmeDataList
				}
				return value;
			},
			mmeStatusFmt(value, nodeInfo, rowIndex) {
				var row = {
					s1siglinkserverlist: nodeInfo.serverList,
					mmeEnable: nodeInfo.mmeEnable,
					mmeStatus: nodeInfo.mmeStatus,
					mme_pool_1: nodeInfo.mmePool1,
					mme_pool_2: nodeInfo.mmePool2
				};

				return mmeStatusFormatterTopo(value, row, rowIndex);
			},
			mmesStatusFmt(str){
				var mmelist = eval('('+str+')');
				var mmeStatus,textVal,value,hasDisconn=false,hasConn=false;
				mmelist.map(function(item){
					if (item.status == '1'){
						hasConn = true;
					}else if(item.status == '0'){
						hasDisconn = true
					}
				})
				
				if (hasDisconn == true){
					if (hasConn == false) {
						mmeStatus ='offStatusCls';
						textVal = 'Disconnected'
					}else{
						mmeStatus ='offOrOnStatusCls';
						textVal = 'Connected'
					}
				}else {
					mmeStatus ='';
					textVal = 'Connected'
				}
				
				value = "<span class='"+ mmeStatus +"'>" + textVal +"</span>"
				return value;
			},
			parseMMEOnNum(str){
				var onNum = [],
					list = [];
				if(str) {
					list = eval('('+str+')');
				}else {
					list = [];
				}
				list.map((item,index)=>{
					if(item.status == '1'){
						onNum.push(item)
					}
				})
				return onNum.length
			},
			parseMME(str) {
				if(str) {
					return eval('('+str+')');
				}else {
					return [];
				}
			},

			mmeIPfmt(value, nodeInfo) {
				var ips = [];

				if(nodeInfo.mmePool1) {
					ips.push(nodeInfo.mmePool1);
				}
				if(nodeInfo.mmePool2) {
					ips.push(nodeInfo.mmePool2);
				}
				if(ips.length==0) {
					ips.push(nodeInfo.serverList);
				}

				return ips.join(', ');
			},
			alarmFmt(value, row) {
				return alarmInfoFormatter(value, row);
			},
			sasEnableFmt(value, row) {
				var vm = this,
					enable = row.sasEnable
					str = '';

				if(enable == '--') {
					str = '--';
				}else if(enable == 'on') {
					str = '<i class="el-icon el-icon-status-enable"></i>' + enable;
				}else {
					str = '<i class="el-icon el-icon-status-disable"></i>' + enable;
				}
				
				return str;
			},
			sasStateFmt(value, row) {
				var vm = this,
					state = row.sasState,
					str = '',
					codes = {
						Unregistered: 'offline',
						Registered: 'registed',
						Granted: 'grated',
						Authorized: 'authed'
					};

				if(value) {
					str = '<i class="el-icon el-icon-topo-enb margin-right-5 ' + codes[value] + '"></i>' + value;
				}

				return str;
			},
			rsrpFmt(nodeInfo) {
				return showRedAccordingRSRP(nodeInfo.RSRP0, nodeInfo) + ' ' + showRedAccordingRSRP(nodeInfo.RSRP1, nodeInfo);
			},
			statusTypeChange(code) {
				var vm = this,
					codes = {
						deviceStatus: ['on','off'],
						activeStatus: ['yes','no'],
						sasStatus: ['Unregistered','Registered','Granted','Authorized']
					};
				
				['deviceStatus','activeStatus','sasStatus'].map(function(item) {
					if(item == code) {
						vm.statusForm[item] = codes[item];
					}else {
						vm.statusForm[item] = []
					}
				});
				
				vm.$nextTick(function(){
					vm.setStatus();
				});
			},
			statusChange(code) {
				var vm = this;
				
				vm.statusType = code;
				
				['deviceStatus','activeStatus','sasStatus'].map(function(item) {
					if(item != code) {
						vm.statusForm[item] = []
					}
				});

				vm.$nextTick(function(){
					vm.setStatus();
				});
			},
			/**
			* 双击列表行 自动定位topo中节点（如果在tops中）
			* @param row{object}: 列表行数据
			* @param evt{event}：鼠标事件
			**/
			rowdblclick(row,evt) {
				var vm = this;
				var code = row.cpe_code||row.serial_number,
					valid = this.allNodes.map(function(row){
						return row.code;
					}).includes(code);
				
				var filterNodes = vm.filterNodesByStatus(vm.repeatedHideNodes),
					filterCodes = filterNodes.map(function(item){
						return item.code
					});
				var isInHidden = filterCodes.includes(code);
				// topo中有该节点时，高亮居中显示
				if((valid || isInHidden) && row.latitude && row.longitude){
					if(isInHidden) {
						var node = filterNodes.filter(function(item){
							return item.code == code;
						})[0];
						// 检测是否为隐藏节点，如果是，移到顶部
						var codeKey = node.lat + '-' + node.lon;
						var repeatedNodes = vm.repeatedMap[codeKey]||'';
						if(repeatedNodes.length) {
							vm.repeatData = vm.repeatedHideNodes.filter(function(item){
								return repeatedNodes.includes(item.code) && filterCodes.includes(item.code);
							});
							vm.allNodes.map(function(item){
								var latlon = item.lat + '-' + item.lon
								if(latlon == codeKey) {
									var copyRow = Object.assign({},item);
									vm.repeatData.unshift(copyRow);
								}
							});
							var index = '';
							vm.repeatData.map(function(item, idx){
								if(item.code == code) index = idx;
							});

							vm.moveTop(node, index);
						}
					}

					globMap.setView([row.latitude-0, row.longitude-0], 3);

					setTimeout(function(){
						highlightNode(code)
						vm.selectedCode = code;
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

				vm.enbQueryParams.search_text = stxt;
			},
			/**
			* 设备列表查询
			* @param text{string}: 检索文本
			**/
			queryCpeDevice(text) {
				var vm = this,
					stxt = (text||'').trim();

				vm.cpeQueryParams.search_text = stxt;
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

				vm.groupIds = ids;
				// 刷新topo和统计信息数据
				vm.getTopoInfos(ids);
				vm.refresh_topo_Statistics(ids);
			},
			refreshTopoData() {
				var vm = this;

				vm.getTopoInfos(vm.groupIds);
			},
			// 状态控制设置
			setStatus() {
				var vm = this;

				Object.assign(vm.form, vm.statusForm);
				//vm.cancelSet();
			},
			// 取消设置关闭浮层
			cancelSet() {
				document.querySelector('#map').click();
				document.querySelector('#map').click();
			},
			nodeHasStatus(node) {
				var vm = this,
					deviceStatus = vm.statusForm.deviceStatus||[],
					activeStatus = vm.statusForm.activeStatus||[],
					sasStatus = vm.statusForm.sasStatus||[],
					hasStatus = false;

				// 设备类型和状态过滤
				if(deviceStatus.length) {
					deviceStatus.includes(node.online) && (hasStatus=true);
				}else if(activeStatus.length) {
					activeStatus.includes(node.active) && (hasStatus=true);
				}else if(sasStatus.length) {
					sasStatus.includes(node.sasState) && (hasStatus=true);
				}
				
				return hasStatus;
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
							if(vm.nodeHasStatus(node)) {// 节点在状态里时
								if(vm.repeatedMap[latlonKey].hasTop) {// 已经有top节点
									vm.repeatedMap[latlonKey].push(code);
									repeatedSNList.push(code);
									vm.repeatedHideNodes.push(node);
								}else {
									vm.repeatedMap[latlonKey].hasTop = true;
								}
							}else {// 节点不在状态里时
								vm.repeatedMap[latlonKey].push(code);
								repeatedSNList.push(code);
								vm.repeatedHideNodes.push(node);
							}
						}else {
							// 第一个节点是否满足状态过滤条件
							if(vm.nodeHasStatus(node)) {
								vm.repeatedMap[latlonKey] = [];
								// 设置已有top节点
								vm.repeatedMap[latlonKey].hasTop = true;
							}else {
								vm.repeatedMap[latlonKey] = [code];
								repeatedSNList.push(code);
								vm.repeatedHideNodes.push(node);
							}
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
					vm.repeatedMap = {};
					vm.repeatedHideNodes = [];
					
					var nodes = res.rows || res.data.rows,
						cpeNodes = [];

					// 节点数据格式规范化 -- 应对变化的接口数据格式
					nodes = transformNode(nodes);
					nodes = vm.initRepeated(nodes);
					vm.enbNodes = nodes;

					// 获取cpe节点
					if(writableMap['CODE_CPE_MONITOR'] != undefined){
						params.device_type = 'CPE';
						axios.post(url,stringify(params)).then(function(res){
							var cpeNodes = res.data.rows;
							
							// 节点数据格式规范化 -- 应对变化的接口数据格式
							cpeNodes = vm.transformCPE(cpeNodes);
							cpeNodes = vm.initRepeated(cpeNodes);
							vm.cpeNodes = cpeNodes;
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
					
					if(isNaN(node.latitude) || [null,undefined].includes(node.latitude)) {
						lat = '';
					}
					if(isNaN(node.longitude) || [null,undefined].includes(node.longitude)) {
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
							sn: node.serial_number,
							alarm: node.alarm,
							sasEnable: node.sas_enable,
							sasState: node.sas_state,
							relaEnb: node.rela_enb,
							RSRP0: node.RSRP0,
							RSRP1: node.RSRP1
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
					vm.cpeDeviceURL = '${ctx}/cell/topo/getDeviceInfoList.action';
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
					cellCode = row.small_cell_code||row.cpe_code,
					serialNumber = row.serial_number;
			
				vm.gpsDeviceType = row.small_cell_code?'enb':'cpe';
				vm.curRow = row;
		    	vm.gpsDlg({
					lat: row.latitude,
					lon: row.longitude,
					cellCode: cellCode,
					sn: row.serial_number,
					height: row.height,
					mechanical_downtilt: row.mechanical_downtilt,
					vertical_3dB_beam_width: row.vertical_3dB_beam_width,
					horizontal_azimuth: row.horizontal_azimuth,
				});
    	    },
			modifyInfoGPS() {
				var vm = this,
					form = vm.infoForm;
				
				vm.gpsDeviceType = form.type == 'enb'?'enb':'cpe';
				vm.gpsDlg({
					lat: form.lat,
					lon: form.lon,
					cellCode: form.type == 'enb'?form.cellCode:form.code,
					sn: form.type == 'enb'?form.code:form.sn,
					height: form.height,
					mechanical_downtilt: form.mechanical_downtilt,
					vertical_3dB_beam_width: form.vertical_3dB_beam_width,
					horizontal_azimuth: form.horizontal_azimuth,
				});
			},
			/**
			* 菜单点击事件
			* @param row{object}: 设备列表行数据
			**/
			gpsDlg(row) {
				var vm = this;
				
				vm.gpsdgVisible = true;
				vm.curDeviceCode = row.cellCode;
				vm.curSerialNumber = row.sn;
				vm.curMacAddress = row.sn;
				vm.$nextTick(function(){
					vm.$refs.gpsform.resetFields();
					Object.assign(vm.gpsForm,{
						lat: row.lat,
						lon: row.lon,
						height: row.height,
						mechanical_downtilt: row.mechanical_downtilt,
						vertical_3dB_beam_width: row.vertical_3dB_beam_width,
						horizontal_azimuth: row.horizontal_azimuth,
					});
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
						setType: 'topo',
						operator_code: operator_code,
						height: vm.gpsForm.height,
						mechanical_downtilt: vm.gpsForm.mechanical_downtilt,
						vertical_3dB_beam_width: vm.gpsForm.vertical_3dB_beam_width,
						horizontal_azimuth: vm.gpsForm.horizontal_azimuth,
					};
				// 保存设备经纬度信息
				vm.$refs.gpsform.validate(function(r){
					if(r) {
						axios.post(url,stringify(params)).then(function(res){
							if(res.data['success']){
								vm.gpsdgVisible = false;

								vm.$refs.enbDevice.refresh();
								if(writableMap['CODE_CPE_MONITOR'] != undefined){
									vm.$refs.cpeDevice.refresh();
								}

								var codes = vm.allNodes.map(function(item){return item.cellCode;});
								if(codes.includes(vm.curDeviceCode)) {
									
									if(vm.curDeviceCode == vm.infoForm.cellCode) {
										Object.assign(vm.infoForm, {
											lon: vm.gpsForm.lon,
											lat: vm.gpsForm.lat,
											height: vm.gpsForm.height,
											mechanical_downtilt: vm.gpsForm.mechanical_downtilt,
											vertical_3dB_beam_width: vm.gpsForm.vertical_3dB_beam_width,
											horizontal_azimuth: vm.gpsForm.horizontal_azimuth,
										})
									}

									var ids = vm.$refs.group.getChecked().join(',');
									vm.getTopoInfos(ids);
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

				nodes = nodes.filter(function(item){
					return item.lat != 0 || item.lon != 0;
				});

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
			* 根据设置面板过滤节点
			* @param nodes{array}: 所有节点
			**/
			filterNodesByStatus(nodes) {
				var vm = this,
					deviceType = vm.statusForm.deviceType||[]
					deviceStatus = vm.statusForm.deviceStatus||[],
					activeStatus = vm.statusForm.activeStatus||[],
					sasStatus = vm.statusForm.sasStatus||[];

				// 设备类型和状态过滤
				if(deviceStatus.length) {
					nodes = nodes.filter(function(node){
						var online = node.online;

						return deviceStatus.includes(online);
					});
				}else if(activeStatus.length) {
					nodes = nodes.filter(function(node){
						var active = node.active;

						if(node.type == 'enb') {
							return activeStatus.includes(active);
						}else {
							return false;
						}
					});
				}else if(sasStatus.length) {
					nodes = nodes.filter(function(node){
						var sas = node.sasState;

						return sasStatus.includes(sas);
					});
				}else {
					return [];
				}
				
				nodes = nodes.filter(function(node){
					var type = node.type;

					return deviceType.includes(type);
				});
				
				return nodes;
			},
			isFitStatus(node) {
				var vm = this,
					nodes = [],
					deviceType = vm.statusForm.deviceType||[]
					deviceStatus = vm.statusForm.deviceStatus||[],
					activeStatus = vm.statusForm.activeStatus||[],
					sasStatus = vm.statusForm.sasStatus||[];
				
				nodes.push(node);
				// 设备类型和状态过滤
				if(deviceStatus.length) {
					nodes = nodes.filter(function(node){
						var online = node.online;
	
						return deviceStatus.includes(online);
					});
				}else if(activeStatus.length) {
					nodes = nodes.filter(function(node){
						var active = node.active;
	
						if(node.type == 'enb') {
							return activeStatus.includes(active);
						}else {
							return true;
						}
					});
				}else if(sasStatus.length) {
					nodes = nodes.filter(function(node){
						var sas = node.sasState;
	
						return sasStatus.includes(sas);
					});
				}else {
					return [];
				}
				
				nodes = nodes.filter(function(node){
					var type = node.type;
	
					return deviceType.includes(type);
				});
				
				return nodes.length>0;
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
								if(cpe && !existedKeys.includes(cpe.code) && vm.isFitStatus(cpe)) {
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
				
				vm.closeMesure();

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
					url = '${ctx}/map/{z}/{x}/{y}.png';
					// 设置中心位置和放大倍数
					map = L.map('map',{maxZoom: 12,minZoom: 4}).fitBounds(bounds); // .setMaxBounds([[-360,-360],[360,360]])
				}else {
					// 设置中心位置和放大倍数
					map = L.map('map',{maxZoom: 18,minZoom: 4}).fitBounds(bounds); // .setMaxBounds([[-360,-360],[360,360]])
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

					drawSignal();
					drawPie({
						selector: base.selector,
						radius: base.radius,         // 外半径
						minRadius: base.minRadius,       // 内半径
						angles: base.angles,         // 弧度
						direct: base.direct*1 + (base.index - 1)*120 // 方向
					});
				});// 事件绑定
				globMap.on('moveend',function(ev){ // 移动结束后重新计算
					vm.filterRender(resObj.enb, globMap,true);
				});
			},
			closeMesure() {
				$('.measure-close.el-icon-close').click();
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

					//slider.style.top = top+'px';
				}

				slider.classList.add('loading');
				if(bool) slider.classList.add('show');
				else slider.classList.remove('show');

				setTimeout(function(){
					slider.classList.remove('loading');
				},1000);

				vm.$refs.enblist.advanceQuery();
				vm.enbQueryParams.search_text = '';
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
				var el = document.querySelector('#mainpage');

				if(navTabsEnable === true) {
					el = document.querySelector('#topo_container');
				}

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
			},
			hidePanel() {
				var vm = this;

				vm.infoShow = false;
				vm.repeatShow = false;
			},
			moveTop(row, index) {
				var vm = this;

				if(index) {
					var tar = vm.repeatData[0];

					vm.repeatData.splice(index,1);
					vm.repeatData.unshift(row);
					vm.replaceTop(row, tar, index);
				}
			},
			replaceTop(node, tar, n) {
				var vm = this;

				var index = -1;
				vm.prevNodes.map(function(item, idx){
					if(item.code == tar.code) index = idx;
				});
				// 替换缓存节点 -- 被移至top的节点，替换原top节点
				if(index>-1) vm.prevNodes.splice(index,1, node);

				var idx = -1;
				vm.repeatedHideNodes.map(function(item, i){
					if(item.code == node.code) idx = i;
				});
				// 替换隐藏节点 -- 原top位置节点，替换至隐藏节点队列中
				if(idx>-1) vm.repeatedHideNodes.splice(idx,1,tar);

				var latlonKey = tar.lat + '-' + tar.lon;
				var nodeIndex = vm.repeatedMap[latlonKey].indexOf(node.code);
				if(nodeIndex>=0) vm.repeatedMap[latlonKey].splice(nodeIndex, 1, tar.code);

				var allIndex = -1;
				vm.allNodes.map(function(item, index){
					if(item.code == tar.code) allIndex = index;
				});
				if(allIndex>-1) vm.allNodes.splice(allIndex,1);

				vm.allNodes.push(node);
				vm.filterRender(vm.allNodes, globMap);
			},
			removeTop(tar) {
				var vm = this,
					latlonKey = tar.lat + '-' + tar.lon,
					node = vm.repeatedMap[latlonKey][0];
				// 将隐藏节点移至顶部显示
				if(node) {
					vm.prevNodes.push(node);

					var idx = -1;
					vm.repeatedHideNodes.map(function(item, i){
						if(item.code == node.code) idx = i;
					});
					if(idx>-1) vm.repeatedHideNodes.splice(idx,1);

					vm.repeatedMap[latlonKey].splice(0, 1);

					vm.allNodes.push(node);
					vm.filterRender(vm.allNodes, globMap);
				}
			},

			resetSiteStatus() {
				var vm = this;

				vm.deviceDetailShow = false;
				vm.topolistShow = false;
				vm.siteModifyHidden = false;
				vm.addSiteDeviceShow = false;
				vm.siteModifyShow = false;
				vm.siteInfoShow = false;
				vm.siteAddHidden = false;
				document.querySelector('#site_map').click();
			},
			addSite() {
				var vm = this;

				vm.siteType = 'add';
				vm.resetSiteStatus();
				vm.siteInfoShow = true;
				vm.siteAddHidden = false;
				Object.assign(vm.siteAddForm, {
					siteName: '',
					latitude: '',
					longitude: '',
					sns: '',
					locationConsistent: false,
				});
				vm.siteAddDeviceType = ['eNB'];
				vm.$refs.addTb.clearSelection();
			},
			siteOptClick(row, evt) {
				var vm = this,
					topoShow = false,
					isNotSBS7 = !['sBS72030','sBS71400'].includes(row.module_type);

				if(['eNB'].includes(row.stationType)) {
					var paramsCode={
							smallCellCode: row.small_cell_code
						},
						showFlagList = [];
		
					$.ajax({
						type: 'POST',
						url: '${ctx}/cell/cpeinfos/getENBOperationItem.action',
						data: paramsCode,
						async: false,
						dataType: 'json',
						success: function(data){
							var operationData = data;
							if(operationData.Maintenance && operationData.Maintenance.length > 0 ){
								operationData.Maintenance.map((item)=>{
									showFlagList[item] = true;
								});
							}
							if(operationData.Actions && operationData.Actions.length > 0 ){
								operationData.Actions.map((item)=>{
									showFlagList[item] = true;
								})
							}
							if(operationData.others && operationData.others.length > 0 ){
								operationData.others.map((item)=>{
									showFlagList[item] = true;
								})
							}
						},
					});

					if(showFlagList.CODE_ENB_DISTRIBUTED && isNotSBS7) {
						topoShow = true;
					}
				}else if(['gNB'].includes(row.stationType)) {
					if(row.product != 'BaiBNQ') {
						topoShow = true;
					}
				}else if(['GSM'].includes(row.stationType)) {
					if(row.product == 'BSC') {
						topoShow = true;
					}
				}

				vm.siteMenus = [
					{label: '<%=rb.getString("XinXi")%>', code: 'view', cls: 'el-icon el-icon-operation-details', row: row},
					{label: 'TOPO', code: 'topo', cls: 'el-icon el-icon-operation-topo', row: row, show: topoShow},
					{label: '<%=rb.getString("SheZhi")%>', code: 'setting', cls: 'el-icon el-icon-operation-settings', row: row},
					{label: '<%=rb.getString("ShanChu")%>', code: 'delete', cls: 'el-icon el-icon-operation-delete', row: row}
				];

				vm.$nextTick(function(){
					document.body.click();
					vm.$refs.siteMenu.show(evt);
				})
			},
			menuClick(row) {
				var vm = this,
					code = row.code,
					rowData = row.row,
					actions = {
						view: vm.viewSiteDevice,
						topo: vm.toTOPOList,
						setting: vm.modifySite,
						delete: vm.delSiteDevice,
					}

				if(typeof(actions[code]) == 'function') {
					vm.resetSiteStatus();
					vm.siteInfoShow = true;
					actions[code](rowData);
				}
			},
			handerClose() {
				this.$refs.siteMenu.hide();
			},
			querySite(txt) {
				var vm = this;

				vm.siteQuery.siteName = txt;
				vm.siteQuery.rd = Math.random();
			},
			siteCurrentChange(row, oldRow) {
				var vm = this;

				vm.siteType = 'view';
				vm.currentSiteRow = row;
				vm.viewQuery.siteName = row.siteName;
				vm.resetSiteStatus();
				vm.siteInfoShow = true;
				vm.siteAddHidden = false;
			},
			viewQueryClick(txt) {
				var vm = this,
					trimTxt = txt.trim();

				vm.viewQuery.search_text = trimTxt;
				vm.viewQuery.rd = Math.random();
			},
			addQueryClick(txt) {
				var vm = this,
					trimTxt = txt.trim();

				vm.addQuery.searchText = trimTxt;
			},
			deviceToSiteQueryClick(txt) {
				var vm = this,
					trimTxt = txt.trim();

				vm.deviceToSiteQuery.searchText = trimTxt;
			},
			addSelectionChange(s) {
				var vm = this,
					list = s.map(item=>{
						return item.serial_number;
					});

				vm.siteAddForm.sns = list.join(',');
			},
			deviceToSiteSelectChange(s) {
				var vm = this,
					list = s.map(item => {
						return item.serial_number
					});

				vm.deviceToSiteForm.sns = list.join(',');
			},
			addSiteSubmit() {
				var vm = this,
					url = '${ctx}/site/addSiteInfo.action',
					params = {};

				Object.assign(params, vm.siteAddForm);

				vm.$refs.addForm.validate(function(r){
					if(r) {
						axios.post(url, stringify(params)).then((res) => {
							var data = res.data;

							if(data.success == true) {
								vm.siteInfoShow = false;
								vm.$refs.siteList.refresh();
								vm.getSiteList();

								vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type:'success'
								})
							}else {
								vm.$message.error(data["message"])
							}
						})
					}
				})
			},
			modifySite(row, evt) {
				var vm = this;

				if(row.serial_number) {
					Object.assign(vm.siteModifyForm,{
						serialNumber: row.serial_number,
						cellCode: row.small_cell_code,
						siteName: '',
						latitude: row.latitude,
						longitude: row.longitude
					})
				}else {
					Object.assign(vm.siteModifyForm,{
						serialNumber: '',
						cellCode: '',
						siteName: row.siteName,
						latitude: row.latitude,
						longitude: row.longitude
					})
				}

				vm.siteModifyShow = true;
				vm.siteModifyHidden = false;
				if(evt) evt.stopPropagation();
			},
			toInfoModifyGPS() {
				var vm = this,
					row = vm.detailForm;

				Object.assign(vm.siteModifyForm,{
					serialNumber: row.code,
					cellCode: row.cellCode,
					siteName: '',
					latitude: row.lat,
					longitude: row.lon
				});

				vm.siteModifyShow = true;
				vm.siteModifyHidden = false;
				vm.deviceDetailShow = false;
			},
			modifySiteSubmit() {
				var vm = this,
					url = '${ctx}/site/updateSiteInfo.action';

				vm.$refs.modifyForm.validate(function(r){
					if(r) {

						if(vm.siteModifyForm.serialNumber) {// save device latlon
							vm.setSiteDeviceLoc(vm.siteModifyForm);
						}else {// save site latlon
							axios.post(url, stringify(vm.siteModifyForm)).then((res)=>{

								vm.siteModifyShow = false;
								vm.$refs.siteList.refresh();
								vm.getSiteList();
							})
						}
					}
				})
			},
			delSite(row, evt) {
				var vm = this,
					url = '${ctx}/site/deleteSiteInfo.action',
					params = {
						siteName: row.siteName
					};

				evt.stopPropagation();

				vm.$confirm('<%=rb.getString("QueDingShanChuRenWu")%>', '<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
				}).then(() => {
					axios.post(url, stringify(params)).then((res)=>{
						
						vm.$refs.siteList.refresh();
						vm.getSiteList();
					})
				}).catch(()=>{})
			},
			modifyPickOnMAP() {
				var vm = this;

				vm.siteModifyHidden = true;
				globSiteMap.off('click', vm.modifyClickEvent).on('click', vm.modifyClickEvent);
				document.querySelector('#site_map').style.cursor = 'crosshair';
			},
			modifyClickEvent(evt) {
				var vm = this;
				
				vm.siteModifyForm.latitude = evt.latlng.lat.toFixed(6);
				vm.siteModifyForm.longitude = evt.latlng.lng.toFixed(6);
				vm.siteModifyHidden = false;
				globSiteMap.off('click', vm.modifyClickEvent);
				document.querySelector('#site_map').style.cursor = '';
			},
			addPickOnMap() {
				var vm = this;

				vm.siteAddHidden = true;
				globSiteMap.off('click', vm.addPickClickEvent).on('click', vm.addPickClickEvent);
				document.querySelector('#site_map').style.cursor = 'crosshair';
			},
			addPickClickEvent(evt) {
				var vm = this;
				
				vm.siteAddForm.latitude = evt.latlng.lat.toFixed(6);
				vm.siteAddForm.longitude = evt.latlng.lng.toFixed(6);
				vm.siteAddHidden = false;
				globSiteMap.off('click', vm.addPickClickEvent);
				document.querySelector('#site_map').style.cursor = '';
			},
			viewSiteDevice(row) {
				var vm = this,
					tRow = vm.thansformRow(row);

				Object.assign(vm.detailForm, tRow);
				vm.deviceDetailShow = true;
			},
			delSiteDevice(row) {
				var vm = this,
					params = {
						siteName: row.sub_station_name,
						serialNumber: row.serial_number
					},
					url = '${ctx}/site/deleteDeviceFromSite.action';

				var confirmStr = '<%=rb.getString("QueDingShanChuRenWu")%>'
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					axios.post(url, stringify(params)).then((res)=>{
						var data = res.data;

						if(data.success == true) {
							vm.$refs.viewTb.refresh();
						}
					})
				}).catch(() => {})
			},
			setSiteDeviceLoc(form) {
				var vm = this;

				$.ajax({
					url: '${ctx}/cell/topo/setLocationInfo.action',
					type: 'post',
					dataType: 'json',
					data: {
						longitude: form.longitude,
						latitude: form.latitude,
						setType: 'topo',
						cell_code: form.cellCode,
						operator_code: operator_code
					},
					success: function(data){
						if(data['success']){
							vm.siteModifyShow = false;
							vm.$refs.viewTb.refresh();
						}else{
							
						}
					}
				})
			},
			toAddSiteDevice() {
				var vm = this;

				vm.deviceToSiteForm.siteName = vm.currentSiteRow.siteName;
				vm.deviceToSiteType = ['eNB'];
				vm.addSiteDeviceShow = true;
				vm.topolistShow = false;

				vm.$nextTick(function(){
					vm.$refs.deviceToSiteTb.clearSelection();
				})
			},
			addSiteDeviceSubmit() {
				var vm = this,
					url = '${ctx}/site/addDeviceToSite.action';

				vm.$refs.siteDeviceFm.validate(function(r){
					if(r) {
						axios.post(url, stringify(vm.deviceToSiteForm)).then((res)=>{

							vm.addSiteDeviceShow = false;
							vm.$refs.viewTb.refresh();
							vm.getSiteList();
						})
					}
				})
			},
			deviceToSiteTypeChange(val) {
				var vm = this;

				vm.$refs.deviceToSiteTb.clearSelection();
			},
			siteAddDeviceTypeChange(val) {
				var vm = this;

				vm.$refs.addTb.clearSelection();
			},
			toTOPOList(row) {
				var vm = this,
					rowData = row.row;

				vm.listOrTopoType = 'list';
				vm.topolistShow = true;

				if(['eNB'].includes(row.stationType)) {
					vm.euUrl = '${ctx}/cell/nxp/queryEUInfos.action';
					vm.ruUrl = '${ctx}/cell/nxp/queryRUInfos.action';
				}else if(row.stationType == 'gNB') {
					vm.euUrl = '${ctx}/gnb/gnbMonitor/getEUInfos.action';
					vm.ruUrl = '${ctx}/gnb/gnbMonitor/getRUInfos.action';
				}else if(row.stationType == 'GSM') {
					vm.btsUrl = '${ctx}/site/getBSCTupoInfos.action';
				}

				vm.queryParamsEURU.smallCellCode = row.small_cell_code;
				vm.queryParamsEURU.type = row.stationType;
			},

			tabClick(tab) {
				var vm = this;

				if(tab.name == 'site') {
					vm.siteListURL = '${ctx}/site/getSiteInfosList.action';
					vm.showSlider(null,false);
					/* 重置其他行箭头状态 */
					Array.from(document.querySelectorAll('.row-op')).map(function(item){
						item.classList.remove('el-icon-arrow-left');
						item.classList.add('el-icon-arrow-right');
					});
					vm.$nextTick(function(){
						vm.getSiteList();
					})
				}
			},
			getSiteList() {
				var vm = this,
					url = '${ctx}/site/getSiteInfosList.action',
					params = {};

				axios.get(url, stringify(params)).then(function(res){
					var nodeList = res.data;

					vm.allSiteLatlonNodes = vm.transformSiteNode(nodeList);
					vm.initSiteTopo();
				});
			},
			transformSiteNode(nodeList) {
				var vm = this;
				
				nodeList = nodeList.map(function(node){
					var lat = node.latitude,
						lon = node.longitude,
						contains = [];

					if(isNaN(node.latitude) || [null,undefined].includes(node.latitude)) {
						lat = '';
					}
					if(isNaN(node.longitude) || [null,undefined].includes(node.longitude)) {
						lon = '';
					}

					if(node.gsmCount && vm.siteDeviceType.includes('gsm')) contains.push('gsm');
					if(node.enbCount && vm.siteDeviceType.includes('enb')) contains.push('enb');
					if(node.gnbCount && vm.siteDeviceType.includes('gnb')) contains.push('gnb');

					return {
						code: node.siteName,
						olat: node.latitude,
						olon: node.longitude,
						lat: lat,
						lon: lon,
						type: 'site',
						contains: contains,
						gsmCount: node.gsmCount,
						enbCount: node.enbCount,
						gnbCount: node.gnbCount
					};
				});

				// 过滤无经纬度的节点
				nodeList = nodeList.filter(function(item){
					return item.lat && item.lon;
				});

				return nodeList;
			},
			initSiteTopo() {
				var vm = this,
					nodes = vm.allSiteLatlonNodes || [];

				if(globSiteMap) globSiteMap.remove();

				var url = 'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png';
				// 处理基站数据格式
				var resObj = vm.proccessNodes(nodes);
				// 获取要展示的区域，fitBounds会自动调整合理显示
				var bounds = vm.getViewBounds(resObj);
				
				if(offlineMapEnable) {
					url = '${ctx}/map/{z}/{x}/{y}.png';
					// 设置中心位置和放大倍数
					siteMap = L.map('site_map',{maxZoom: 12,minZoom: 4}).fitBounds(bounds);
				}else {
					// 设置中心位置和放大倍数
					siteMap = L.map('site_map',{maxZoom: 18,minZoom: 4}).fitBounds(bounds); 
				}

				globSiteMap = siteMap;
				
				// 关联背景地图资源
				L.tileLayer(url,{
					attribution: ''
				}).addTo(siteMap);
				// 绘制 Site 图层
				vm.createSiteLayer(resObj.enb, siteMap);

				// 事件绑定
				siteMap.on('zoomend',function(ev){ // 缩放结束后重新计算
					vm.siteFilterRender(resObj.enb, siteMap);
				});// 事件绑定
				siteMap.on('moveend',function(ev){ // 移动结束后重新计算
					vm.siteFilterRender(resObj.enb, siteMap, true);
				});
			},
			rowdblclickSite(row, evt) {
				var code = row.siteName,
					valid = this.allSiteLatlonNodes.map(function(row){
						return row.code;
					}).includes(code);

				// topo中有该节点时，高亮居中显示
				if(valid && row.latitude && row.longitude){
					globSiteMap.setView([row.latitude-0, row.longitude-0], 3);
					setTimeout(function(){
						highlightSiteNode(code)
					},500)
				}
			},
			getSiteLimitedNodes(nodes, bufferEnable) {

				return nodes;
			},
			siteFilterRender(nodes, siteMap, bufferEnable) {
				var vm = this,
					bounds = siteMap.getBounds(),
					sw = bounds._southWest,
					ne = bounds._northEast,
					zoom = siteMap.getZoom(),
					threshold = this.limit;

				// 删除历史节点图层
				siteMap.eachLayer(function(layer){
					// 根据options中的特性，匹配节点和连线的layer，然后删除
					if(layer.options && (layer.options.type == 'site'||layer.options.fromField == 'site')) {
						siteMap.removeLayer(layer);
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
					
					vm.createSiteLayer(inviewNodes, siteMap, bufferEnable);
				}else {
					vm.createSiteLayer(nodes, siteMap, bufferEnable);
				}
			},
			createSiteLayer(nodes, siteMap, bufferEnable) {
				var vm = this;
				// 判断可视区域内的节点是否超过阀值
				nodes = this.getSiteLimitedNodes(nodes, bufferEnable);
				// 创建以code为主键的检索队列
	        	var nodesLookup = L.GeometryUtils.arrayToMap(nodes, 'code');
				var nodeOptions = vm.getSiteOptions();// 生成Site的Options
	            // 渲染Site节点
				var nodesLayer = new L.MarkerDataLayer(nodesLookup, nodeOptions);
				siteMap.addLayer(nodesLayer);
			},
			getSiteOptions() {
				var sizeFunction = new L.LinearFunction([1, 16], [253, 48]);
				var vm = this,
					options = {
						type: 'site',
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
							draggable: false,
							fill: false,
							stroke: false,
							weight: 0,
							color: '#A0A0A0'
						},
						filter: function (record) {
							return true;
						},
						setIcon: function (record, options) {// 自定义节点图标
							var html = ['<div class="node-item-leaf">',
											'<span class="node-code">' + record.code + '</span>',
											'<i class="el-icon el-icon_tpopo_site blue-bg" style="font-size: 30px;"></i>',
											'<div class="site-device-tag">',
												record.contains.includes('gsm') && vm.isGSMEnable?'<i class="el-icon el-icon_tpopo_2G"></i>':'',
												record.contains.includes('enb')?'<i class="el-icon el-icon-topo-enb"></i>':'',
												record.contains.includes('gnb')?'<i class="el-icon el-icon_tpopo_5G"></i>':'',
											'</div>',
										'</div>'].join('');

							var $html = $(html);

							var directFlights = L.Util.getFieldValue(record, 'direct_flights');
							var size = sizeFunction.evaluate(directFlights);
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
								if(isMeasuring) {
									measureObj.clickHandler(evt);
							
									evt.stopPropagation();

									return;
								}
								highlightSiteNode(record.code);
								// 点击节点时，弹出详情浮层页
								vm.siteType = 'view';
								vm.resetSiteStatus();
								vm.siteInfoShow = true;
								vm.siteAddHidden = false;
								vm.currentSiteRow = {
									siteName: record.code,
									latitude: record.lat,
									longitude: record.lon
								};
								vm.viewQuery.siteName = record.code;
							});

							layer.off('mouseover').on('mouseover', function (evt) {
								if(isMeasuring) return;
								layer.openPopup();
							});
							
							$(window).resize();
							// 点击节点弹出明细层
							layer.bindPopup(
								$(vm.popoverSiteInfo(record)).wrap('<div/>').parent().html(),
								{autoPan: false, keepInView: true}
							);
						}
					};

				return options;
			},
			popoverSiteInfo(node) {
				var vm = this,
					content = [
						'<div class="panelDefault" style="width: 270px;height: 160px;border: none;">',
							'<div class="el-icon el-icon-close node-info-close" onclick="cancelSet()"></div>',
							'<div class="tabsContentDiv">',
								'<div style="position: relative;border-bottom: 1px solid #eee;padding: 5px;font-weight: bold;">{code}</div>',
								'<div class="infoTab" style="display: block;position: relative;">',
									'<ul class="node-ul">',
									'<li class="space-between"><span class="node-label"><%=rb.getString("WeiDu")%></span> {lat}</li>',
									'<li class="space-between"><span class="node-label"><%=rb.getString("JingDu")%></span> {lon}</li>',
									vm.isGSMEnable?'<li class="space-between"><span class="node-label">GSM Count</span> {gsmCount} </li>':'',
									'<li class="space-between"><span class="node-label">eNB Count</span> {enbCount} </li>',
									'<li class="space-between"><span class="node-label">gNB Count</span> {gnbCount} </li>',
									'<ul>',
								'</div>',
							'</div>',
						'</div>'
					];

				return content.join(' ').evaluate(node);
			},
			changeListOrTopo(type) {
				var vm = this;

				vm.listOrTopoType = type;

				if(type == 'topo') {
					vm.$nextTick(function(){
						vm.initDeviceTopo();
					});
				}
			},
			domToImage() {
				var vm = this,
					origin = document.querySelector('#root'),
					options = {
						width: origin.offsetWidth,
						height: origin.offsetHeight,
						quality: 1
					};

				domtoimage.toPng(origin, options).then(function(dataUrl){
					vm.dataURL = dataUrl
				}).catch(error=>{
					console.log('oops, somethine went wrong!', error)
				})
			},
			initDeviceTopo() {
				var vm = this,
					params = { 
						smallCellCode: vm.queryParamsEURU.smallCellCode
					};
				
				axios.post('${ctx}/gnb/gnbMonitor/getTOPOInfo.action',stringify(params)).then(function(response){
					var data = response.data;
					if(data){
						//添加节点数据
						var nodeDataArray = data.nodeDataArray;
						//添加连接线数据
						var linkDataArray = data.linkDataArray;

						var nodes = nodeDataArray.map(function(item){
								return {
									id: item.key, 
									label: item.text, 
									color: {
										background: '#fff', 
										border: '#67d972',
										highlight: {border: '#ffc300'} 
									}
								}
							}),
							edges = linkDataArray.map(function(item){
								return {
									from: item.from, 
									to: item.to, 
									font: {background: 'lime'}
								}
							}),
							container = document.getElementById('device_vis_topo'),
							data = {
								nodes: nodes,
								edges: edges
							},
							options = {
								nodes: {
									borderWidth: 2,
									shape: 'box',
									margin: 10
								},
								edges: {
									smooth: {
										type: 'cubicBezier',
										forceDirection: 'vertical',
										roundness: 0.5
									}
								},
								layout: {
									hierarchical: {
										direction: 'UD'
									}
								},
								physics: {
									stabilization: false
								}
							};
		
						var network = new vis.Network(container, data, options);
					}
				}).catch(function(error){});
				
				//initTopo(params.smallCellCode);
			},

			siteCheckAllChange(val) {
				var vm = this;

				vm.isIndeterminate = false;
				vm.siteDeviceType = val? (vm.isGSMEnable?['gsm','enb','gnb']:['enb','gnb']) : [];

				vm.initSiteTopo();
			},
			siteCheckChange() {
				var vm = this,
					length = vm.siteDeviceType.length;

				vm.siteCheckAll = length == 3;
				vm.isIndeterminate = length < 3 && length > 0;
				
				vm.getSiteList();
			},

			thansformRow(node) {
				var vm = this;
				var enable = node.mme_enable == 1? true:false,
					mmePool_1 = node.mme_pool_1?node.mme_pool_1:'',
					mmePool_2 = node.mme_pool_2?node.mme_pool_2:'',
					serverList = node.s1siglinkserverlist?node.s1siglinkserverlist:'',
					lat = node.latitude,
					lon = node.longitude;
				if(isNaN(node.latitude) || [null,undefined].includes(node.latitude)) {
					lat = '';
				}
				if(isNaN(node.longitude) || [null,undefined].includes(node.longitude)) {
					lon = '';
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
					alarm: node.alarm,
					sasEnable: node.sas_enable,
					sasState: node.sas_state,
					relaCpeList: node.relaCpeList,
					NEW_MME_STATUS: node.new_mme_status
				};
			},
		},
		mounted() {
			var vm = this;
			// 获取sas开关状态
			axios.post('${ctx}/cell/topo/getSasEnableStatus.action',stringify({operator_code: operator_code})).then(function(res){
				if(res && res.data) {
					vm.sasEnable = !res.data.success || writableMap.CODE_ENB_MONITOR !== true;
				}
			});

			vm.initOverView();

			if(window.topoOverViewTimer) {
				clearInterval(window.topoOverViewTimer);
			}
			window.topoOverViewTimer = setInterval(vm.initOverView, 600000);

			$('.flex-bt-cls').on('click', function(evt) {
				var target = $(evt.target);
				$('.flex-bt-cls').removeClass('selected');
				target.addClass('selected');
			})
		}
	});

	//画图： 图表由 节点、文字、线组成
	//1、stroke: 边框颜色；  2、 margin: 边框间距,   margin: new go.Margin(10,20,30,40) 外边距;   3、fill: 背景颜色；
	//1、TextBlock: 创建文本；   2、Shape: 创建图形；  3、 Node:节点（结合文本与图形）；  4、Links 连线
	function initTopo(commonSn){
		// 创建图表
		var $ = go.GraphObject.make;
		//绑定DOM元素
		myDiagram = 
			$(go.Diagram, 'device_vis_topo', 
				{
					isReadOnly: true,					
					// 画布的位置设置：居中显示内容, 不可拖动画布
	            	contentAlignment: go.Spot.Center,
	            	// 启用Ctrl-Z和Ctrl-Y撤销重做功能
					'undoManager.isEnabled': false,
					//去掉节点 点击时的边框颜色
					nodeSelectionAdornmentTemplate:
						$(	go.Adornment,
							'Aiuto',
							$(go.Shape, 'Rectangle',{fill:'white',stroke: null})		
						),
					//树形布局排列方式，从上到下（0，90，180，270） 、每层间距,
					layout: $(go.TreeLayout, {angle: 90, layerSpacing: 66}) 					
				}
			);
		
		//新建节点
		myDiagram.nodeTemplate = 
			$(go.Node, 'Auto',
				//突出显示 鼠标滑过、离开
				{
					selectionAdorned: false,
					selectionChanged: allChanged,
					//鼠标滑过显示 设备名称
					mouseEnter: mouseEnter,
					mouseLeave: mouseLeave
				},
				//节点禁止拖动
				{movable: false},
				
				//设置节点形状：圆角矩形
				$(go.Shape, 'RoundedRectangle',
					//设置大小、边框大小、颜色、背景色、鼠标手势
					{width: 120, height: 60, strokeWidth: 2, margin: new go.Margin(0,0,0,20), cursor: 'grab', name: 'SHAPE'},
					//将节点数据nodeDataArray   .color与节点背景色建立联系
					//绑定背景色
					new go.Binding('fill', 'bgColor'),
					//绑定边框色
					new go.Binding('stroke', 'borderColor'),
				),
				//设置文本节点
				$(go.TextBlock, textStyle(),
					//设置文本样式：大小，是否换行，margin
					{ wrap: go.TextBlock.WrapFit, name: 'TEXT'},
					//将TextBlock.text 绑定到 Node.data.text
					new go.Binding('text', 'text'),
					new go.Binding('cursor', 'cursor')),
				//添加 tooltip，显示节点对应的基站编码
				{toolTip:
					$("ToolTip",
						$(go.TextBlock,{margin: 6},
							new go.Binding('text','serialnumber', 
								function(sn){ 
									return '<%=rb.getString("KPISheBei")%>: ' + sn;
								}							
							))		
					)					
				} 
			);
		
		function allChanged(node){
			if(node.isSelected){
				if(node.part.data.outOfContact == 'true'){
				    selectionAdornment.adornedObject = node;
				    node.addAdornment('Radial',selectionAdornment);
			   }else{
				   var oldnode = selectionAdornment.adornedPart;
				   if(oldnode) oldnode.removeAdornment('Radial');
				   selectionAdornment.adornedObject = null;
			   } 
			}else{
				 var oldnode = selectionAdornment.adornedPart;
				 if(oldnode) oldnode.removeAdornment('Radial');
				 selectionAdornment.adornedObject = null;
			}
		};
		
		var selectionAdornment = 
			$(go.Adornment, 'Spot',
				$(go.Panel, 'Auto',
					$(go.Shape, {fill: null, stroke:'#DCDFE6', strokeWidth: 0}),
					$(go.Placeholder)
				), 
				$('Button',
					{ alignment: go.Spot.Top, alignmentFocus: go.Spot.Left, 
						width: 140, cursor: 'pointer', padding: go.Margin.parse('0 0 20 10'), 
						'ButtonBorder.fill': '#FFFFFF',
						'ButtonBorder.stroke': '#E9E9E9',
						'_buttonFillOver': '#EDF6FF',
						'_buttonStrokeOver': '#E9E9E9',
						'_buttonFillFocus': '#EDF6FF',
						'_buttonStrokeFocus': '#E9E9E9',
						click: function(e, obj){
							if(obj.part.data.outOfContact == 'true'){
							    curNodeData = obj.part.data;
							    gNBTopoConfig.delNodeClick();
						    }else{
							   return;
						    }
						}
					},
					
					$(go.TextBlock, '<%=rb.getString("ShanChu") %>',
							{margin: 6,stroke: '#333333'},		
					)							
				)	
			)
				
		//虚线 实线	 连接	
		var templmap = new go.Map(), color  = '#FA5555';
		var defaultTemplate = 
			$(go.Link,
				$(go.Shape, { stroke: color, strokeWidth: 2})
			)
		var dashedTemplate = 
			$(go.Link,
				$(go.Shape, { stroke: color, strokeWidth: 2, strokeDashArray: [6, 3]})			
			)
		
		templmap.add('',defaultTemplate);
		templmap.add('dashed',dashedTemplate);
		myDiagram.linkTemplateMap = templmap; 
		
		//设置线条，暂无箭头
		myDiagram.linkTemplate = 
			$(go.Link,
				//线条连接样式：直线
				{curve: go.Link.Bezier},
				//线的连接形状
				$(go.Shape,
					{strokeWidth: 2, stroke: "#707070"}	
				)
			);
		
		 // 定义图形上的文字风格		 
	    function textStyle() {
	        return {         
	        	//文本颜色
	            stroke: "#333333",
	            font: "bold 12px normal"
	        }
	    };
	    //鼠标滑过修改背景颜色
		function mouseEnter(e,obj){			
			var shape = obj.findObject('SHAPE');
				if(obj.data.hoverColor == ''){
					shape.fill = '#F3F3F3';
				}else{
					shape.fill = obj.data.hoverColor;
				}
		};
		//鼠标离开还原色值
		function mouseLeave(e,obj){
			var shape = obj.findObject('SHAPE');
			shape.fill = obj.data.bgColor;
			
		};
	    
	    //获取TOPO 图数据
	    setTimeout(function(){
		    var params = { smallCellCode: commonSn};
		    axios.post('${ctx}/gnb/gnbMonitor/getTOPOInfo.action',stringify(params)).then(function(response){
		   		var data = response.data;
				if(data){
					//添加节点数据
					var nodeDataArray = data.nodeDataArray;
					
					//添加连接线数据
					var linkDataArray = data.linkDataArray;
	
					//新建关系图: 通过节点数据和关系数组完成关系图
					myDiagram.model = new go.GraphLinksModel(nodeDataArray,linkDataArray);	
				}
			}).catch(function(error){}) 

			/* var linkDataArray = [
				{from: 0, to: -1},
				{from: -1, to: 11, category:'dashed'},
				{from: -1, to: 13, category:'dashed'},
				{from: -1, to: 14, category:'dashed'}
			]; 
			var nodeDataArray = [		    
				{key: 0, text: 'BBU', bgColor: '#F2F6FF', borderColor:'#4D84FF', hoverColor: '#DCEBFE',},			
				{key: -1, text: 'HUB1', bgColor: '#F5FFF9', borderColor:'#67D972', hoverColor: '#D8F3E5', serialnumber: '1202000024019APP0061'},
				{key: 11, text: 'RRU1', bgColor: '#F3F3F3', borderColor:'#DCDFE6',  hoverColor: '#F3F3F3',serialnumber: '1202000024019APP0062', outOfContact: 'true'},
				{key: 13, text: 'HUB1-1', bgColor: '#F3F3F3', borderColor:'#DCDFE6', hoverColor: '#F3F3F3', serialnumber: '1202000024019APP0063', outOfContact: 'true'},
				{key: 14, text: 'HUB1-2', bgColor: '#F3F3F3', borderColor:'#DCDFE6',  hoverColor: '', serialnumber: '1202000024019APP0064', outOfContact: 'true'}
			];

			myDiagram.model = new go.GraphLinksModel(nodeDataArray,linkDataArray);*/
	    },500);	   
	   
	}

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
		}else {
			vm.repeatedMap[latlonKey] = [];
		}

		vm.filterRender(vm.allNodes, globMap);
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
				lon = node.longitude,
				meDowntilt = node.mechanical_downtilt,
				elDowntilt = node.electronic_downtilt,
				vertical3dB = node.vertical_3dB_beam_width,
				horizontal_azimuth = node.horizontal_azimuth,
				radius = 0,
				minRadius = 0,
				height = node.height;

			if(height && meDowntilt && elDowntilt && vertical3dB) {
				radius = ( height / Math.tan(2*Math.PI/360 * (meDowntilt*1 + elDowntilt*1 - vertical3dB/2)) ).toFixed(2);
				minRadius = ( height / Math.tan(2*Math.PI/360 * (meDowntilt*1 + elDowntilt*1 + vertical3dB/2)) ).toFixed(2);
			}
			
			if(isNaN(node.latitude) || [null,undefined].includes(node.latitude)) {
				lat = '';
			}
			if(isNaN(node.longitude) || [null,undefined].includes(node.longitude)) {
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
				alarm: node.alarm,
				sasEnable: node.sas_enable,
				sasState: node.sas_state,
				relaCpeList: node.relaCpeList,
				NEW_MME_STATUS: node.new_mme_status,
				pci: node.phycellid,
				height: height,
				direct: horizontal_azimuth,
				radius: radius,
				minRadius: minRadius,
				angles: 120,
				meDowntilt: meDowntilt,
				elDowntilt: elDowntilt,
				vertical3dB: vertical3dB,

				mechanical_downtilt: meDowntilt,
				electronic_downtilt: elDowntilt,
				vertical_3dB_beam_width: vertical3dB,
				horizontal_azimuth: horizontal_azimuth,
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
			if($dom.text() == code) {
				$dom.addClass('selected');
				$dom.parent().addClass('selected');
			}else {
				$dom.removeClass('selected');
				$dom.parent().removeClass('selected');
			}
		});
	}
	
	
	function highlightSiteNode(code){
		$('.node-code').each(function(idx,item){
			var $dom = $(item);
			if($dom.text() == code) {
				$dom.addClass('site-selected');
				$dom.parent().addClass('site-selected');
			}else {
				$dom.removeClass('site-selected');
				$dom.parent().removeClass('site-selected');
			}
		});
	}
	
	function showRepeatedList(dom,event) {
		var ctn = $(dom).parents('.leaflet-marker-icon'),
			codeKey = $(dom).attr('key'),
			code = $(dom).attr('code');

		event.stopPropagation();

		// 根据控制状态过滤节点
		var filterNodes = topovm.filterNodesByStatus(topovm.repeatedHideNodes).map(function(item){
			return item.code
		});

		var repeatedNodes = topovm.repeatedMap[codeKey]||'';

		topovm.infoShow = false;
		topovm.repeatShow = true;
		topovm.repeatData = topovm.repeatedHideNodes.filter(function(item){
			return repeatedNodes.includes(item.code) && filterNodes.includes(item.code);
		});

		var row = topovm.allNodes.filter(function(item){
					return item.code == code;
				  })[0];
		
		if(row) topovm.repeatData.unshift(row);

		topovm.repeatedQueryText = '';
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
					draggable: !topovm.sasEnable,
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

						Object.assign(topovm.gpsmvoeForm, {
							lat: latlng.lat.toFixed(3),
							lon: latlng.lng.toFixed(3),
							olatlng: olatlng,
							cell_code: code,
							operator_code: operator_code,
							target: target
						});
						topovm.gpsmoveVisible = true;
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

					if(record.type == 'enb') {
						var selectedCode = topovm.selectedCode,
							isCurNode = selectedCode == record.code,
							selectCls = isCurNode? 'expand':'',
							cellIndex = codeCellMap[selectedCode],
							cellCtnExpand = isCurNode && cellIndex > 0,
							sectorCls = cellCtnExpand ? 'expand':'',
							cell1Cls = cellCtnExpand && cellIndex == '1' ? 'select expand':'',
							cell2Cls = cellCtnExpand && cellIndex == '2' ? 'select expand':'',
							cell3Cls = cellCtnExpand && cellIndex == '3' ? 'select expand':'';
						// codeCellMap
						$html.prepend('<span class="cell-code" style="display: none;">' + record.cellCode + '</span>');
						console.log('selected code: ', selectedCode, selectCls, selectedCode == record.cellCode)
						var singalHtml = [  '<div class="sector-ctn ' + selectCls + '">',
												'<div class="sector ' + sectorCls + '">',
													'<div onclick="blurCell()" class="sector-close">X</div>',
												'</div>',
												'<div class="cell-sector-ctn">',
												'<div class="cell-sector cell-1 ' + cell1Cls + '" style="width: 50px;height: 50px;top: -25px;left: -25px;"></div>',
												//'<div class="cell-sector cell-2 ' + cell2Cls + '" style="width: 50px;height: 50px;top: -25px;left: -25px;"></div>',
												//'<div class="cell-sector cell-3 ' + cell3Cls + '" style="width: 50px;height: 50px;top: -25px;left: -25px;"></div>',

												'<div onclick="cellClick(this, event)" cell="1" class="hover-square cell-1"></div>',
												//'<div onclick="cellClick(this, event)" cell="2" class="hover-square cell-2"></div>',
												//'<div onclick="cellClick(this, event)" cell="3" class="hover-square cell-3"></div>',
												'</div>',
											'</div>'
											].join(' ');
						
						if(record.radius) $html.prepend(singalHtml);

					}else $html.prepend('<span class="cell-code" style="display: none;">' + record.cellCode + '</span>');
					
					// 生成重复GPS节点信息
					var latlonKey = record.lat+'-'+record.lon,
						repeatedNodes = topovm.repeatedMap[latlonKey]||'',
						listText = '';

					// 根据控制状态过滤节点
					var filterNodes = topovm.filterNodesByStatus(topovm.repeatedHideNodes).map(function(item){
						return item.code
					});

					if(repeatedNodes) {
						repeatedNodes = repeatedNodes.filter(function(code){
							return filterNodes.includes(code);
						});
					}

					if(repeatedNodes.length) {
						repeatedNodes.map(function(item){
							listText += '<span class="repeated-item">' + item + '</span>';
						});
						listText = '<div class="repeated-list">' + listText + '</div>';
						
						listText += '<span class="repeated-count" key="'+latlonKey+'" code="'+record.code+'" onclick="showRepeatedList(this,event)">+'+ (repeatedNodes.length+1) +'</span>';

						$html.prepend('<span class="repeated-nodes" onclick="event.stopPropagation()">'+ listText +'</span>');
					}
					
					// 连接状态信息
					if(record.online == 'off'){
						$html.addClass('offline');
					}
					// sas 状态
					if(topovm.statusType == 'sasStatus' && record.sasState && ['Unregistered','Registered','Granted','Authorized'].includes(record.sasState)) {
						var codes = {
							Unregistered: 'offline',
							Registered: 'registed',
							Granted: 'grated',
							Authorized: 'authed'
						};
						$html.addClass(codes[record.sasState]);
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
						if(isMeasuring) {
							measureObj.clickHandler(evt);
							
							evt.stopPropagation();

							return;
						}
						// 节点高亮
						if(record.type != 'enb') highlightNode(record.code);
						
						topovm.selectedCode = record.code;
						let nodeCtn = $('.node-code').filter(function() {
							return $(this).text() == record.code;
						})[0];
						
						if(nodeCtn) {
							$('.sector-ctn').removeClass('expand');
							$('.sector-ctn', nodeCtn.parentNode).addClass('expand');
						}

						base.direct = (record.direct || 0.1) - 15;
						cellOptions.direct = (record.direct || 0.1) - 15;
						base.radius = record.radius;
						cellOptions.radius = record.radius;
						base.minRadius = record.minRadius;
						cellOptions.minRadius = record.minRadius;
						
						drawSignal();
						drawPie({
							selector: base.selector,
							radius: base.radius,         // 外半径
							minRadius: base.minRadius,   // 内半径
							angles: base.angles,         // 弧度
							direct: base.direct*1 + (base.index - 1)*120          // 方向
						});

						if($(evt.originalEvent.target).hasClass('alarm-info')){
							setTimeout(function(){
								$('#map').click();
							},0);
							setTimeout(function(){
								jumpToAliveAlarm(record.alarmLevel, record.code);
							},15);
						}
						$('.repeated-list').hide();

						// 显示详情
						showDeviceInfo(record, evt);
					});

					layer.off('mouseover').on('mouseover', function (evt) {
						if(isMeasuring) return;
						//layer.openPopup();
					});
					
					$(window).resize();
					// 点击节点弹出明细层
					layer.bindPopup($(showNodeInfo(record)).wrap('<div/>').parent().html(),{autoPan: false, keepInView: true});
				}
			};
        return options;
    }
	function showDeviceInfo(item, evt) {
		Object.assign(topovm.infoForm, item);
		topovm.infoShow = true;
		topovm.repeatShow = false;
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
            var params = {
                alarm_severity: '', 
                unread: '',
                search_text: sn,
            };
    		eventAllBus.$emit("gomenupage","8000","","8000",params,function(){
                alarmViewVue.activeName = 'alarmView';
				alarmViewVue.resetQueryParams(params);
            });

			if(alarmViewVue) {
				alarmViewVue.activeName = 'alarmView';
				alarmViewVue.resetQueryParams(params);
			}
    	}catch(e){}
    }
	/**
	* 格式化节点详情信息
	* @param node{object}：节点
	**/
	function showNodeInfo(node){
		var nodeInfo = $.extend({},node);
		var str = [
				'<div class="panelDefault" style="width: 270px;height: 160px;border: none;">',
					'<div class="el-icon el-icon-close node-info-close" onclick="cancelSet()"></div>',
					'<div class="tabsContentDiv">',
						'<div style="position: relative;border-bottom: 1px solid #eee;padding: 5px;font-weight: bold;">eNB Information</div>',
						'<div class="infoTab" style="display: block;position: relative;">',
							'<ul class="node-ul">',
							'<li><span class="node-label"><%=rb.getString("DianYuanBianMa")%><%=rb.getString("MaoHao")%></span> {code}</li>',
							'<li><span class="node-label"><%=rb.getString("SheBeiZhuangTai")%></span> {online}</li>',
							'<li><span class="node-label"><%=rb.getString("ShiFouJiHuo")%></span> {active}</li>',
							'<li><span class="node-label"><%=rb.getString("GaoJingJiBie")%></span> {alarm}</li>',
							writableMap['CODE_ADVANCE_SAS']!=undefined?'<li><span class="node-label"><%=rb.getString("SASZhuangTai")%></span> {sasState}</li>':'',
							'<ul>',
						'</div>',
					'</div>',
				'</div>'
			];
		
		if(node.type=='cpe') {
			str = [
				'<div class="panelDefault" style="width: 270px;height: 160px;border: none;">',
					'<div class="el-icon el-icon-close node-info-close" onclick="cancelSet()"></div>',
					'<div class="tabsContentDiv">',
						'<div style="position: relative;border-bottom: 1px solid #eee;padding: 5px;font-weight: bold;">CPE Information</div>',
						'<div class="infoTab" style="display: block;position: relative;">',
							'<ul class="node-ul">',
							'<li><span class="node-label"><%=rb.getString("DianYuanBianMa")%><%=rb.getString("MaoHao")%></span> {sn}</li>',
							'<li><span class="node-label"><%=rb.getString("SheBeiZhuangTai")%></span> {online}</li>',
							writableMap['CODE_ADVANCE_SAS']!=undefined?'<li><span class="node-label"><%=rb.getString("SASZhuangTai")%></span> {sasState}</li>':'',
							'<li><span class="node-label">eNB SN</span> {relaEnb} </li>',
							'<ul>',
						'</div>',
					'</div>',
				'</div>'
			];
			
			nodeInfo.relaEnb = node.relaEnb;
		}

		nodeInfo.active = node.active == 'yes'? '<i class="el-icon el-icon-status-active online margin-right-5"></i><%=rb.getString("JiHuo")%>':'<i class="el-icon el-icon-status-active inactive margin-right-5"></i><%=rb.getString("QuJiHuo")%>';
		nodeInfo.online = node.online == 'on'? '<i class="el-icon el-icon-status-conn-on margin-right-5"></i><%=rb.getString("ZaiXian")%>':'<i class="el-icon el-icon-status-conn-off margin-right-5"></i><%=rb.getString("LiXian")%>';
		nodeInfo.sasState = sasStatusFormatter(nodeInfo.sasState, nodeInfo);
		nodeInfo.alarm = alarmPopoverFormatter(nodeInfo.alarm, nodeInfo);

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
			nodeInfo.mmeFormat = mmeStatusFormatterTopo(nodeInfo.mmeStatus, rowData);
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
	function mmeStatusFormatterTopo(value, rowData, rowIndex){
		if (value == null || value == "") {
			return null;
		}
		
		if (value == "1") {
			value = "<div class='mmeConnItem' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='MME'><span class='el-icon el-icon-status-MME'></span></div>"
					+ "<div class='mmeDetails MMEDetail'>MME IP : "+rowData.s1siglinkserverlist + "</br>MME Status : <span class='mmeStauts'></span></div>";
		} else if (value == "0") {
			value = "<div class='mmeDisconnItem' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='MME'><span class='el-icon el-icon-status-MME'></span></div>"
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
					ret = ret + "<div class='mmeConnItem' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' type='MME1'><span class='el-icon el-icon-status-MME1'></span></div>"; 
				}else if(mmeArr.length == 2 && "mme1" == mmeArr[0] && mmeArr[1] == "0"){
					ret = ret + "<div class='mmeDisconnItem' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' type='MME1'><span  class='el-icon el-icon-status-MME1'></span></div>";
				}else if(mmeArr.length == 2 && "mme2" == mmeArr[0] && mmeArr[1] == "1"){
					ret = ret + "<div class='mmeConnItem' onmouseover='toMMEDetail(this,1)' onmouseout='hideMMEDetail()' style='margin-left:10px;' type='MME2'><span  class='el-icon el-icon-status-MME2'></span></div>";
				}else if(mmeArr.length == 2 && "mme2" == mmeArr[0] && mmeArr[1] == "0"){
					ret = ret + "<div class='mmeDisconnItem' onmouseover='toMMEDetail(this,0)' onmouseout='hideMMEDetail()' style='margin-left:10px;' type='MME2'><span class='el-icon el-icon-status-MME2'></span></div>"; 
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
	function sasStatusFormatter(value, row) {
		var state = row.sasState,
			str = '',
			codes = {
				Unregistered: 'offline',
				Registered: 'registed',
				Granted: 'grated',
				Authorized: 'authed'
			};

		if(value) {
			str = '<i class="el-icon el-icon-topo-enb margin-right-5 ' + codes[value] + '"></i>' + value;
		}

		return str;
	}
	function alarmPopoverFormatter(value, row) {
		var str = '',
			alarmObj =value;
		
		if(alarmObj) {
			var keys = ['31001','31002','31003','31004'],
				alarmCount = 0,
				levels = {
					31001: 'Critical',
					31002: 'Major',
					31003: 'Minor',
					31004: 'Warning',
				},
				types = [],
				type = '';

			keys.map(function(key){
				if(alarmObj[key]) {
					alarmCount += alarmObj[key]*1;
					types.push(levels[key]);
				}
			});

			type = types[0];

			if(alarmCount) {
				str = '<span class="el-icon el-icon-menu-alarm margin-right-5 '+ type +'"></span>'
						+type+'<span style="color: blue;cursor: pointer;" onclick="jumpToAliveAlarmEnb(&quot;' + row.code + '&quot;)">('+(alarmCount*1)+')</span>';
			}
		}

		return str;
	}
	function alarmInfoFormatter(value, row) {
		var str = '',
			alarmObj =value;
		
		if(alarmObj) {
			var keys = ['31001','31002','31003','31004'],
				keycls = [],
				alarmCount = 0,
				levels = {
					31001: 'Critical',
					31002: 'Major',
					31003: 'Minor',
					31004: 'Warning',
				},
				types = [],
				type = '';

			keys.map(function(key){
				if(alarmObj[key]) {
					keycls.push(levels[key]);
					alarmCount += alarmObj[key]*1;
					types.push(levels[key]);
				}
			});

			type = types[0];

			if(alarmCount) {
				str = '<span class="alarm-circle margin-right-5 '+ type +'">' + alarmCount + '</span>'+ type;
			}
		}

		return str;
	}
	function jumpToAliveAlarmEnb(sn) {
    	try{
    		eventAllBus.$emit("gomenupage","8000","","8000",{search_text: sn});

			if(alarmViewVue) {
				alarmViewVue.activeName = 'alarmView';
				alarmViewVue.resetQueryParams('');
				alarmViewVue.search_text = sn;
				alarmViewVue.params_alarm.searchText = sn;
			}
    	}catch(e){}
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
	
	function showRedAccordingRSRP(value, rowData, rowIndex) {
        var maxValue = highVal,
            minValue = lowVal;
        
        var ret ;
        if ( value ){
            if (value < minValue) {
                ret = "<span class='el-icon el-icon-signal signal-low' style='display:flex;align-items:center;'>" + value + "</span>";
                
            } else if (value > maxValue){
                ret = "<span class='el-icon el-icon-signal signal-high' style='display:flex;align-items:center;'>" + value + "</span>";
            }else {
                ret = "<span class='el-icon el-icon-signal signal-normal' style='display:flex;align-items:center;'>" + value + "</span>";
            }
        }else {
            ret = value || ''
        }
        
        return ret;
    }

	var direct = 0.1;
	
	var base = {
			selector: '.sector',
			radius: 0,     // 外半径
			minRadius: 0,   // 内半径
			angles: 120,     // 弧度
			direct: direct,  // 方向
			index: 1,
		},
		cellOptions = {
			radius: 0,
			minRadius: 0,
			angles: 120,
			direct: direct,
			translate: 'translate(12.5px, 12.5px)'
		},
		codeCellMap = {

		};

	function drawSignal() {
		// cell-1
		drawPie({
			selector: '.cell-sector-ctn .cell-1',
			radius: cellOptions.radius,        // 外半径
			minRadius: cellOptions.minRadius,  // 内半径
			angles: cellOptions.angles,        // 弧度
			direct: cellOptions.direct || 0.1,        // 方向
			translate: cellOptions.translate
		});
		/*
		// cell-2
		drawPie({
			selector: '.cell-sector-ctn .cell-2',
			radius: cellOptions.radius,        // 外半径
			minRadius: cellOptions.minRadius,  // 内半径
			angles: cellOptions.angles,        // 弧度
			direct: cellOptions.direct + 120,  // 方向
			translate: cellOptions.translate
		});
		// cell-3
		drawPie({
			selector: '.cell-sector-ctn .cell-3',
			radius: cellOptions.radius,        // 外半径
			minRadius: cellOptions.minRadius,  // 内半径
			angles: cellOptions.angles,        // 弧度
			direct: cellOptions.direct + 240,  // 方向
			translate: cellOptions.translate
		});
		*/
		
		let squareDirect = cellOptions.direct - 127,
			squareTranslate = cellOptions.translate;
		// 容器大小和方向
		addCSSRule('.hover-square.cell-1', {
			transform: 'rotate(' + squareDirect + 'deg)' + ' ' + squareTranslate + ' !important'
		});
		/*
		// 容器大小和方向
		addCSSRule('.hover-square.cell-2', {
			transform: 'rotate(' + (squareDirect + 120) + 'deg)' + ' ' + squareTranslate + ' !important'
		});
		// 容器大小和方向
		addCSSRule('.hover-square.cell-3', {
			transform: 'rotate(' + (squareDirect + 240) + 'deg)' + ' ' + squareTranslate + ' !important'
		});
		*/
	}

	function blurCell(evt) {
		$('.sector').removeClass('expand');
		document.querySelectorAll('.cell-sector').forEach((item)=>{
			item.classList.remove('expand');
		});

		evt.stopPropagation();
	}

	function cellClick(dom, evt) {
		let target = evt.target,
			cellIndex = target.getAttribute('cell');
		
		let nodeCtn = $('.node-code').filter(function() {
			return $(this).text() == topovm.selectedCode;
		})[0].parentNode;

		codeCellMap[topovm.selectedCode] = cellIndex;
		// 动态绘制小区信号扇面
		base.index = cellIndex - 0; // 更新信号扇面方向
		
		drawPie({
			selector: base.selector,
			radius: base.radius,     // 外半径
			minRadius: base.minRadius,   // 内半径
			angles: base.angles,  // 弧度
			direct: base.direct*1 + (base.index - 1)*120 // 方向
		});
		// 添加expand类，展示信号区域
		document.querySelectorAll('.cell-sector').forEach((item)=>{
			item.classList.remove('expand');
			item.classList.remove('select');
		});
		let curCell = $('.cell-sector.cell-'+cellIndex, nodeCtn);
		curCell.addClass('expand');
		curCell.addClass('select');
		$('.sector', nodeCtn).addClass('expand');

		evt.stopPropagation();
	}
	/**
	 * @param {Number} radius: 外半径
	 * @param {Number} minRadius: 内半径
	 * @param {Number} angles: 角度
	 * @param {Number} direct: 方向
	 **/
	 function drawPie({radius, minRadius, angles, direct, selector}) {
		let points = ['50% 50%', '0 0'],
			residue = (angles*1)%45? (angles*1)%45:45,
			percent = 0
			direct = direct*1 || 45;

		angles = angles*1;
		
		angles > 90 && points.push('100% 0');
		angles > 180 && points.push('100% 100%');
		angles > 270 && points.push('0 100%');
		
		//let percent = (100/2) * (Math.tan(2*Math.PI/360 * residue).toFixed(4)); // tan算出来的是相对半径的占比
		
		percent = (100/2) * (Math.tan(2*Math.PI/360 * residue).toFixed(4)); 
		
		if(angles<=45) {
			points.push(percent + '%' + ' 0');
		}else if(angles<=90) {
			points.push(percent + 50 + '%' + ' 0');
		}else if(angles<=135) {
			points.push('100% ' + percent + '%');
		}else if(angles<=180) {
			points.push('100% ' + (percent + 50) + '%');
		}else if(angles<=225) {
			points.push(100 - percent + '%' + ' 100%');
		}else if(angles<=270) {
			points.push(50 - percent + '%' + ' 100%');
		}else if(angles<=315) {
			points.push('0 ' + (100 - percent) + '%');
		}else if(angles<=360) {
			points.push('0 ' + (50 - percent) + '%');
		}
		
		let path = 'polygon(' + points.join(', ') + ')',
			zoom = 5;
		try{
			zoom = globMap.getZoom();
		}catch(e){}

		let rateRels = {
				0: 0.10,
				1: 0.10,
				2: 0.10,
				3: 0.10,
				4: 0.13,
				5: 0.25,
				6: 0.45,
				7: 0.75,
				8: 1.3,
				9: 2,
				10: 2,
				11: 3,
				12: 3,
				13: 5,
				14: 5,
				15: 8,
				16: 8,
				17: 12,
				18: 12
			},
			rate = rateRels[zoom]*zoom;
		console.log('zoom: ', zoom, rate)

		// 外扇面
		addCSSRule(selector + '::before', {
			'clip-path': path
		});
		// 内扇面
		addCSSRule(selector + '::after', {
			'clip-path': path,
			width: minRadius*rate + 'px',
			height: minRadius*rate + 'px',
			top: 'calc(50% - ' + minRadius*rate/2 + 'px)',
			left: 'calc(50% - ' + minRadius*rate/2 + 'px)'
		});
		let rateRadius = radius*rate;
		rateRadius = rateRadius < 30 ? 30:rateRadius;
		// 容器大小和方向
		addCSSRule(selector, {
			top: '-' + rateRadius/2 +'px',
			left: '-' + rateRadius/2 +'px',
			width: rateRadius + 'px',
			height: rateRadius + 'px',
			transform: 'rotate(' + direct + 'deg)'
		});
	}
	/*
	// 绘制扇面
	drawPie({
		selector: '.sector',
		radius: 400,     // 外半径
		minRadius: 180,  // 内半径
		angles: 120,     // 弧度
		direct: 45       // 方向
	});
	*/
	function addCSSRule(selector, rules, index) {
	    // 创建一个style元素
	    var style = document.createElement('style');
	    
	    // 设置type属性为text/css
	    style.type = 'text/css';
	    
	    // 插入到head中
	    document.head.appendChild(style);
	    
	    // 获取sheet
	    var sheet = style.sheet;
	    
	    // 如果index未提供，则添加到末尾
	    index = index || null;
	    
	    // 如果是CSS规则字符串
	    if (typeof rules === 'string') {
	        // 直接添加
	        sheet.insertRule(selector + ' {' + rules + '}', index);
	    } else { // 如果是一个对象
	        // 遍历对象中的所有属性
	        for (var prop in rules) {
	            if (rules.hasOwnProperty(prop)) {
	                // 将属性和值转换为字符串
	                var rule = prop + ': ' + rules[prop];
	                // 添加到样式表中
	                sheet.insertRule(selector + ' {' + rule + '}', index);
	            }
	        }
	    }
	}
	
</script>

