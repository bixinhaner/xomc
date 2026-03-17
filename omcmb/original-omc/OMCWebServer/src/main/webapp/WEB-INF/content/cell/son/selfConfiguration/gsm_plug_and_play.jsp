<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
	#gsmPlugAndPlayPage {
		position: relative;
		height: 100%;
	}
	#gsmPlugAndPlayPage i.disabled { 
		opacity: 0.6; 
	}
	#gsmPlugAndPlayPage .el-table th { 
		background: #F9F9F9; 
	}
	#gsmPlugAndPlayPage .el-table td { 
		color: #666666; 
	}
	#gsmPlugAndPlayPage .el-table--border th { 
		border-right: 1px solid #E9E9E9; 
	}
	#gsmPlugAndPlayPage .el-table th>.cell { 
		color: #999999; 
		font-weight: normal; 
	}
	#gsmPlugAndPlayPage .el-date-editor .el-range-input { 
		font-size: 12px; 
	}
	#gsmPlugAndPlayPage .el-date-editor .el-range__close-icon { 
		margin-top: -12px; 
		width: 0;
		font-size: 12px;
	}
	#gsmPlugAndPlayPage .el-range-editor.el-input__inner {
		padding: 3px;
	}
	#gsmPlugAndPlayPage .el-tooltip__popper.is-dark { 
		margin: 0 30px 0 80px; 
	}#gsmPlugAndPlayPage .el-picker-panel {
		margin: 5px -130px;
	}
	#gsmPlugAndPlayPage .el-progress-bar { 
		width: 85px; 
	}
	#gsmPlugAndPlayPage .el-progress__text {  
		display: none; 
	}
	#gsmPlugAndPlayPage .addBtnCls { 
		top: 22px; 
	}
	#gsmPlugAndPlayPage .mainContent {
		height: 100%; 
		display: flex; 
		flex-direction: column;
	}
	#gsmPlugAndPlayPage .policyTableList {
		margin-bottom: 10px; 
		border: 1px solid #D5DCEC;
		border-radius: 10px;
	}
	#gsmPlugAndPlayPage .commonToolBarBox .el-query .advanceQuery .el-icon { 
		font-size: 14px;
	}
	#gsmPlugAndPlayPage .commonToolBarBox .el-query .advanceQuery { 
		height: 26px; 
	}
	#gsmPlugAndPlayPage .commonToolBarBox .el-query .advanceQuery .el-input.el-input--small { 
		width: 280px; 
	}
	#gsmPlugAndPlayPage .commonToolBarBox .el-query .advanceQuery .el-input.el-input--small .el-input__inner { 
		height: 26px; 
		line-height: 26px; 
		background-color: #FFFFFF;
		cursor: pointer;
	}
	#gsmPlugAndPlayPage .commonToolBarBox { 
		height: 30px;
	 }
	#gsmPlugAndPlayPage .commonToolBarBox .el-query .advanceQuery .el-icon:before,
	#gsmPlugAndPlayPage .commonIcon:before,
	#gsmPlugAndPlayPage .informationWarpSlide .el-card__header { 
		color: #7A7992; 
	}
	#gsmPlugAndPlayPage .gsmTableTitle { 
		line-height: 30px;
		padding-left: 20px;
	}
	#gsmPlugAndPlayPage .statusText { 
		font-size: 12px;
		color: #666666; 
	}
	#gsmPlugAndPlayPage .statusSize { 
		font-size: 18px; 
		margin-right: 4px; 
		margin-top: 2px; 
	}
	#gsmPlugAndPlayPage .el-icon-status-disable:before { 
		color: #B8C3D9; 
	}
	#gsmPlugAndPlayPage .el-icon-status-enable:before { 
		color: #4ED76E; 
	}
	#gsmPlugAndPlayPage .gsmHorizontalLine {
		cursor: row-resize;
		padding: 2px;
	}
	#gsmPlugAndPlayPage .statusTableList {
		overflow: auto; 
		flex: auto; 
		border: 1px solid #D5DCEC; 
		border-radius: 10px;
	}
	#gsmPlugAndPlayPage .statusTableQuery {
		margin-bottom: 8px;
		padding-right: 10px; 
		justify-content: space-between;
	}
	#gsmPlugAndPlayPage .resultHeadBox .statisticsSuccessDiv{
		height: 14px;
		line-height: 14px;
		display: flex;
		font-size: 14px;
		border-right: 1px solid #DFE2EE;
	}
	#gsmPlugAndPlayPage .operationBtn {
		margin-right: 6px;
	}
	#gsmPlugAndPlayPage .resultHeadBox .statisticsFailDiv{
		height: 14px;
		line-height: 14px;
		display: flex;
		font-size: 14px;
	}
	#gsmPlugAndPlayPage .statisticsFailDiv .el-icon::before{
		color: #E88282;
		content:'\e6fb';
	}
	#gsmPlugAndPlayPage .statisticsFailDiv > div:first-child{
		padding: 0px 10px;
		color: #E88282;
	}
	#gsmPlugAndPlayPage .statisticsFailDiv .el-icon-circle-close:before {
		content: '\e6fb'
	}
	#gsmPlugAndPlayPage .statisticsSuccessDiv .el-icon::before{
		color:#67D972;
	}
	#gsmPlugAndPlayPage .statisticsSuccessDiv > div:first-child{
		padding: 0px 10px;
		color: #67D972;
	}
	#gsmPlugAndPlayPage .statisticsSuccessDiv > div:last-child{
		padding: 0px 10px 0 0;
	}
	#gsmPlugAndPlayPage .executeStatusQuery {
		position: relative;
		padding-right: 10px;
		padding-left: 20px;
	}
	#gsmPlugAndPlayPage .executeStatusQuery .el-query { 
		right: 0; 
	}
	#gsmPlugAndPlayPage .radioBox .el-radio-button__inner { 
		padding: 0 12px; 
		border: none; 
		line-height: 24px; 
		font-size: 12px; 
	}
	#gsmPlugAndPlayPage .radioBox .el-radio-button:first-child .el-radio-button__inner { 
		border-radius: 100px 0 0 100px; 
		border-left: none; 
	}
	#gsmPlugAndPlayPage .radioBox .el-radio-button:last-child .el-radio-button__inner { 
		border-radius: 0 100px 100px 0; 
	}
	#gsmPlugAndPlayPage .radioBox .el-radio-button__orig-radio:checked+.el-radio-button__inner {
		color:var(--main-color);
		background: rgba(var(--main-color-rgba1),0.1);
		border-color:var(--main-color) !important;
		border-radius: 100px;
	}
	#gsmPlugAndPlayPage .radioBox { 
		border: 1px solid #E9E9E9; 
		height: 26px; 
		border-radius: 100px; 
		background: #FFFFFF !important;
	}
	#gsmPlugAndPlayPage .queryGroup { 
		height: 24px; 
	}
	#gsmPlugAndPlayPage .queryGroup .el-input { 
		width: 280px; 
	}
	#gsmPlugAndPlayPage .queryGroup .el-input__inner {
		width: 280px; 
		height: 24px; 
		line-height: 24px;
		padding: 0; 
	}
	#gsmPlugAndPlayPage .queryGroup .el-icon-common-search {
		font-size: 14px; 
	}
	#gsmPlugAndPlayPage .queryGroup .el-icon-common-search:before { 
		color: #7A7992; 
	}
	#gsmPlugAndPlayPage .specialQuery { 
		margin-left: 10px; 
	}
	#gsmPlugAndPlayPage .specialQuery .el-input__inner { 
		width: 200px;
	}
	#gsmPlugAndPlayPage .dataRange {
		width: 280px; 
		margin: 0 10px;
	}
	#gsmPlugAndPlayPage .commonWidth110 .el-input { 
		width: 110px; 
	}
	#gsmPlugAndPlayPage .defaultClass { 
		cursor: pointer; 
	}
	#gsmPlugAndPlayPage .disabledClass { 
		cursor: not-allowed !important; 
		opacity: 0.4; 
	}
	#gsmPlugAndPlayPage .disabledClass:before {
		color: #c0c4cc;  
		cursor: not-allowed !important; 
	}
	#gsmPlugAndPlayPage .statusBoxCls {
		margin-left: 10px;
	}
	#gsmPlugAndPlayPage .runningStatus {  
		width: 90px; 
		background: #E4F1FF;  
		color: #4D84FF; 
	}
	#gsmPlugAndPlayPage .successStatus {  
		width: 76px; 
		background: #EEFFF3;  
		color: #4ED76E;
	}
	#gsmPlugAndPlayPage .skipStatus { 
		width: 90px; 
		background: #FFFDE3;  
		color: #FFAA00;
	}
	#gsmPlugAndPlayPage .failStatus { 
		width: 50px; 
		background: #FEF2F2;  
		color: #FF6D59; 
	}
	#gsmPlugAndPlayPage .commonStatus { 
		height: 24px; 
		line-height: 24px; 
		border-radius: 100px; 
		text-align: center; 
	}
	#gsmPlugAndPlayPage .slide-position-top .el-icon-close {
		font-size: 14px !important;
	}
	#gsmPlugAndPlayPage .informationWarpSlide { 
		width: calc(100% - 20px) !important; 
		height: 218px !important; 
		margin: 0 8px; 
		bottom: 50px !important; 
		left: 2px !important; 
	}
	#gsmPlugAndPlayPage .informationWarpSlide .el-card { 
		border-radius: 0 0 10px 10px; 
	}
	#gsmPlugAndPlayPage .informationWarpSlide .el-card__header span:last-child { 
		right: 4px !important; 
	}
	#gsmPlugAndPlayPage .informationWarpSlide .el-card__header .el-icon-close {  
		font-size: 14px; 
	}
	.gsmPlugPlayAddSlide { 
		width: 100% !important; 
		height: 100% !important; 
		border-radius: 10px; 
	}
	.gsmPlugPlayAddSlide .el-card__header { 
		height: 40px !important; 
		line-height: 40px !important; 
	}
	.newCommonTableBorder {
		border: 1px solid #D5DCEC;
	}
	.gsmDetectionTitle {
		padding: 15px 0;
		display: flex;
		align-items: center;
	}
	.searchCls {
		margin-top: 5px; 
		font-size: 16px; 
	}
	.itemShow .el-tabs__content {
		margin-top: -30px;
	}
	.itemHide .el-tabs__content {
		margin-top: 0px;
	}

	#gsmPlugAndPlayPage .editButton{
		position: absolute;
		right: 20px;
		top: 0px;
		padding:0 10px;
		height:24px;
		background:#F2F9FF;
		border-radius:2px;
		line-height:24px;
		cursor:pointer;
		margin-left:10px;
		border:1px solid #1DA3FC;
		display:inline-block;
	}
	#gsmPlugAndPlayPage .editButton i{
		font-size:14px !important;
	}
	#gsmPlugAndPlayPage .editButton span{
		font-size:12px;
	}
	.gsmDetectCheckDialog .editButton{
		position: absolute;
		right: 60px;
		top: 0px;
		padding:0 10px;
		height:24px;
		background:#F2F9FF;
		border-radius:2px;
		line-height:24px;
		cursor:pointer;
		margin-left:10px;
		border:1px solid #1DA3FC;
		display:inline-block;
	}
	.gsmDetectCheckDialog .editButton i{
		font-size:14px !important;
	}
	.gsmDetectCheckDialog .editButton span{
		font-size:12px;
	}
	.gsmDetectCheckDialog .el-pairgrid-title {
		/*display: none !important;*/
		line-height: 50px !important;
	}
</style>
<!-- 即插即用 -->
<div id="gsmPlugAndPlayPage" class="monitor-ctner" style="overflow: auto; padding-top: 10px; height: calc(100% - 10px)">
	<div class="circleIcon placeholder-bt addBtnCls hidden CODE_PLUG_AND_PLAY" placeholder='<%=rb.getString("XinZeng")%>'>
		<span class="el-icon-circle-add el-icon" @click="gsmAddTaskBtnClick"></span>
	</div>
	<div class='mainContent'>
		<!--gsm 策略列表  :data="tbData"-->
		<el-ctable ref="gsmPolicyList" class='gsmFlexRowItem policyTableList'
			:url="gsmConfigURL" :pagination="false"
			:query-params="gsmPolicyQuery"
			row-key="policyId"
			@row-click="gsmRowClick"
			@load-success="gsmPolicyListSuccess">
			<div slot="toolbar">
				<div class='commonFlex commonToolBarBox commonContent'>
					<div class='commonTextWeight14 gsmTableTitle commonLeft20'><%=rb.getString("CeLueLieBiao")%>({{gsmPolicyNumber}})</div>
					<el-query type="normal" @query="queryPolicyGsm" placeholder='<%=rb.getString("CeLueMingChen") %> / <%=rb.getString("MuBiaoBanBen")%>'></el-query>
				</div>
			</div>
			<el-table-column width="40">
				<template slot-scope="scope">
					<div class="el-icon el-icon-main-more" @click="gsmOptClick(scope.row, event)" v-clickoutside="gsmHanderClose"></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("ShiFouQiYong")%>' prop="switch" width="100" style="position:relative;">
				<template slot-scope="scope">
					<el-switch v-model="scope.row.switch" style="height: 18px;"
						:before-change="gsmSwitchChange"
						:active-value="'1'"
						:inactive-value="'0'"
						active-color="#4D84FF"
						inactive-color="#CFCFCF">
					</el-switch>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>' prop="productName" show-overflow-tooltip="true" width="100"></el-table-column>
			<el-table-column label='<%=rb.getString("CeLueMingChen") %>' prop="policyName" show-overflow-tooltip="true"></el-table-column>
			<el-table-column label='<%=rb.getString("ZhiXingFangShi")%>' prop="executeType" width="200">
				<template slot-scope="scope">
					<div v-if="scope.row.executeType === '1'"><%=rb.getString("eNBShouDongZhiXing") %></div>
					<div v-if="scope.row.executeType === '0'"><%=rb.getString("eNBZiDongZhiXing") %></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("RuanJianShengJi")%>' prop="upgradeEnable" show-overflow-tooltip="true"> 
				<template slot-scope="scope">
					<div v-if='scope.row.upgradeEnable == "0"' class='commonFlex'>
						<span class="el-icon el-icon-status-disable statusSize"></span>
						<span class='statusText'><%=rb.getString("JinYong")%> <span v-show='scope.row.destVersion !="" && scope.row.destVersion !=null && scope.row.destVersion !=undefined'><%=rb.getString("MuBiaoBanBen")%>={{scope.row.destVersion}}</span></span>
					</div>
					<div v-if='scope.row.upgradeEnable == "1"' class='commonFlex'>
						<span class="el-icon el-icon-status-enable statusSize"></span>
						<span class='statusText'><%=rb.getString("QiYong")%> <span v-show='scope.row.destVersion !="" && scope.row.destVersion !=null && scope.row.destVersion !=undefined'><%=rb.getString("MuBiaoBanBen")%>={{scope.row.destVersion}}</span></span>
					</div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("License")%>' prop="licenseEnable" width="200">
				<template slot-scope="scope">
					<div v-if='scope.row.licenseEnable === "0"' class='commonFlex'>
						<span class="el-icon el-icon-status-disable statusSize"></span>
						<span class='statusText'><%=rb.getString("JinYong")%></span>
					</div>
					<div v-if='scope.row.licenseEnable === "1"' class='commonFlex'>
						<span class="el-icon el-icon-status-enable statusSize"></span>
						<span class='statusText'><%=rb.getString("QiYong")%></span>
					</div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("ENBCanShuZiPeiZhi")%>' prop="configEnable" width="200"> 
				<template slot-scope="scope">
					<div v-if='scope.row.configEnable == "0"' class='commonFlex'>
						<span class="el-icon el-icon-status-disable statusSize"></span>
						<span class='statusText'><%=rb.getString("JinYong")%></span>
					</div>
					<div v-if='scope.row.configEnable == "1"' class='commonFlex'>
						<span class="el-icon el-icon-status-enable statusSize"></span>
						<span class='statusText'><%=rb.getString("QiYong")%></span>
					</div>
				</template>
			</el-table-column>
		</el-ctable>
		<div class="gsmHorizontalLine"> </div>
		<!-- 执行状态列表-->
		<el-ctable id="gsmConfigStatusList" height="76%" ref="gsmStatusList" class='gsmFlexRowItem statusTableList'
			:time="6"
			row-key="task_id"
			:url="gsmStatusURL"
			:query-params="gsmStatusQuery"
			@load-success="gsmConfigStatusSuccess">
			<div slot="toolbar">
				<div class='commonFlex statusTableQuery'>
					<div class='commonTextWeight14 gsmTableTitle'><%=rb.getString("ZhiXingZhuangTai")%></div>
					<div class="resultHeadBox commonFlex" style='margin-top: 10px;'>
						<div class="statisticsSuccessDiv">
							<div><span class="el-icon el-icon-circle-success operationBtn"></span><%=rb.getString("ChengGong")%></div>
							<div>{{gsmSuccessCount}}</div>
						</div>
						<div class="statisticsFailDiv">
							<div><span class="el-icon el-icon-circle-close operationBtn"></span><%=rb.getString("ShiBai")%></div>
							<div>{{gsmFailCount}}</div>
						</div>
					</div>
				</div>
				<div class='commonFlex commonToolBarBox executeStatusQuery'>
					<el-radio-group v-model='gsmTabsActive' class='radioBox' @change="gsmTabsChange">
						<el-radio-button label="0"><%=rb.getString("SuoYouRenWu") %></el-radio-button>
						<el-radio-button label="1"><%=rb.getString("RuanJianShengJi") %></el-radio-button>
						<el-radio-button label="2"><%=rb.getString("License") %></el-radio-button>
						<el-radio-button label="3"><%=rb.getString("ENBCanShuZiPeiZhi") %></el-radio-button>
					</el-radio-group>
					<div class='commonFlex statusBoxCls'>
						<div class='queryGroup specialQuery' style='width: 220px;'>
							<el-input v-model='statusSearchText' @keyup.enter.native="queryGsmStatus" placeholder='<%=rb.getString("CeLueMingChen") %> / <%=rb.getString("Title_SheBeiBianMa")%>' style='width: 200px;'></el-input>
							<i @click='queryGsmStatus' class="el-icon el-icon-common-search"></i>
						</div>
						<el-date-picker class='dataRange'
							v-model="gsmTimeRange"
							type="datetimerange"
							value-format="yyyy-MM-dd HH:mm:ss"
							range-separator="一"
							@change="gsmDateChange"
							start-placeholder='<%=rb.getString("KaiShiShiJian")%>'
							end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
						</el-date-picker>
						<el-select placeholder='<%=rb.getString("ChanPinLeiXingBiaoZhi") %>' v-model='gsmStatusQuery.productType' class='commonWidth110'>
							<el-option v-for="item in productTypeList" :label="item.name" :value="item.value"></el-option>
						</el-select>
						<el-select placeholder='<%=rb.getString("ZhuangTai") %>' v-model='gsmStatusQuery.status' class='commonWidth110' style='margin: 0 10px;'>
							<el-option label='<%=rb.getString("SuoYou")%>' value=" "></el-option>
							<el-option label='<%=rb.getString("ChengGong")%>' value="0"></el-option>
							<el-option label='<%=rb.getString("ShiBai")%>' value="1"></el-option>
							<el-option label='<%=rb.getString("JinXingZhong")%>' value="2"></el-option>
							<el-option label='<%=rb.getString("WeiZhiXing")%>' value="3"></el-option>
							<el-option label='<%=rb.getString("TiaoGuo")%>' value="4"></el-option>
						</el-select>
					</div>
				</div>
			</div>
			<!--status: 0-成功， 1-失败，2-进行中，3-未执行（等待）， 4-跳过，  5-部分成功 已去掉-->
			<!--  手动（1）或未执行（3）的任务可以 start;   删除： 等待，失败，成功，部分成功；      重新执行 retry：失败，成功  ;    跳过skip： 可retry ,delete; all: 显示结果页面-->
			<el-table-column width="125" v-if="gsmTabsActive == '0'" :key='6'>
				<template slot-scope="scope">
					<div class='commonFlex'>
						<div>
							<i v-if="(scope.row.status=='0' || scope.row.status=='1' || scope.row.status=='4') && gsmPnpEnable && gsmPolicySwitch == '1'" @click="gsmRestartStatus(scope.row)" :class="[retryLoading==scope.row.policyId  ? 'disabledClass' : 'defaultClass']" class="el-icon el-icon-operation-restart commonIcon defaultClass" title="<%=rb.getString("ChongXinZhiXing")%>"></i>
							<i v-else class="el-icon el-icon-operation-restart commonIcon disabledClass" title='<%=rb.getString("ChongXinZhiXing")%>'></i>
						</div>
						<div class="statusBoxCls" v-show="scope.row.execute_type === '1'">
							<!-- 只有手动模式显示 -->
							<i v-if="scope.row.status=='3' && gsmPnpEnable" @click="gsmStartOperStatus(scope.row)" class="el-icon el-icon-operation-start commonIcon defaultClass" title="<%=rb.getString("ZhiXingNew")%>"></i>
							<i v-else class="el-icon el-icon-operation-start commonIcon disabledClass" title='<%=rb.getString("ZhiXingNew")%>'></i>
						</div>
						<div class="statusBoxCls">
							<i v-if="(scope.row.status=='0' || scope.row.status=='1' || scope.row.status=='3' || scope.row.status=='4') && gsmPnpEnable" @click="gsmDeletStatus(scope.row)" class="el-icon el-icon-operation-delete commonIcon defaultClass" title="<%=rb.getString("ShanChu")%>"></i>
							<i v-else class="el-icon el-icon-operation-delete commonIcon disabledClass" title='<%=rb.getString("ShanChu")%>'></i>
						</div>
						<div class="statusBoxCls">
							<i @click="gsmViewConfigTask(scope.row)" class="el-icon el-icon-operation-info commonIcon" title='<%=rb.getString("XinXi")%>'></i>
						</div>
					</div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("Title_SheBeiBianMa")%>' prop="serial_number" width="240" show-overflow-tooltip="true"></el-table-column>
			<el-table-column label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>' prop="product_type" width="160"></el-table-column>
			<el-table-column label='<%=rb.getString("CeLueMingChen") %>' prop="policy_name" width="200" show-overflow-tooltip="true"></el-table-column>
			<el-table-column label='<%=rb.getString("ZhiXingFangShi")%>' prop="execute_type" width="130">
				<template slot-scope="scope">
					<div v-if="scope.row.execute_type === '1'"><%=rb.getString("eNBShouDongZhiXing") %></div>
					<div v-if="scope.row.execute_type === '0'"><%=rb.getString("eNBZiDongZhiXing") %></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("KaiShiShiJian") %>' prop="start_time" width="150"></el-table-column>
			<el-table-column label='<%=rb.getString("JieShuShiJian") %>' prop="end_time" width="150"></el-table-column>
			<el-table-column label='<%=rb.getString("BuJinDu") %>' prop="execute_procedure" width="210" v-if="gsmTabsActive == '0'" key='execute_procedure'></el-table-column>
			<el-table-column label='<%=rb.getString("ChuShiBanBen") %>' prop="original_version" width="150" key='original_version' v-if="gsmTabsActive == '1'"></el-table-column>
			<el-table-column label='<%=rb.getString("MuBiaoBanBen") %>' prop="target_version" width="150" key='target_version' v-if="gsmTabsActive == '1'"></el-table-column>
			<el-table-column label='<%=rb.getString("LicenseWenJian") %>' prop="lic_file" width="200" key='lic_file' v-if="gsmTabsActive == '2'"></el-table-column>
			<el-table-column label='<%=rb.getString("ZhuangTai")%>' prop="status" width="110">
				<template slot-scope="scope" class="status-info">
					<div class='successStatus commonStatus' v-if="scope.row.status=='0'"><%=rb.getString("ChengGong") %></div>
					<div class='failStatus commonStatus' v-if="scope.row.status=='1'"><%=rb.getString("ShiBai")%></div>
					<div class='runningStatus commonStatus' v-if="scope.row.status=='2'"><%=rb.getString("JinXingZhong")%></div>
					<div class='runningStatus commonStatus' v-if="scope.row.status=='3'"><%=rb.getString("WeiZhiXing") %></div>
					<div class='skipStatus commonStatus' v-if="scope.row.status=='4'"><%=rb.getString("TiaoGuo")%></div>
				</template>
			</el-table-column>
			<el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' prop="failure_reason" show-overflow-tooltip="true" min-width="200"></el-table-column>
		</el-ctable>
		<el-cmenu ref="gsmMenu" :data="gsmMenus" @click="gsmMenuClick"></el-cmenu>
	</div>
	<!-- Add Config Slider -->
	<el-slide ref="gsmSlider" title='<%=rb.getString("XinZeng")%>'
		:url="slideURL"
		:title="slideTile"
		:header='slideHeader'
		:footer="footerShow" class='gsmPlugPlayAddSlide'
		@ok="gsmAddConfig"
		@cancel="gsmCloseConfig">
	</el-slide>

	<!-- gsm execute status 详情 -->
	<el-slide ref="slideView" title="Devices Execute Information" :footer="false" position="bottom" class='informationWarpSlide'
	    height="300px" :modal='modal' @cancel='gsmCancelViewSlide'>
	    	<el-ctable id="exeProgressTable" ref="ctableProgress" :url="taskUrl" :query-params="params_progress" time="6"
				:row-key="'id'" :pagination="false" :rownumber="false">
				<el-table-column label='<%=rb.getString("JinDu")%>' prop="progress">
					<template slot-scope="scope" class="status-info">
						<div v-if="scope.row.progress=='1'"><%=rb.getString("RuanJianShengJi")%></div>
						<div v-else-if="scope.row.progress=='2'"><%=rb.getString("License")%></div>
						<div v-else-if="scope.row.progress=='3'"><%=rb.getString("ENBCanShuZiPeiZhi") %></div>
						<div v-else-if="scope.row.progress=='4'">Cell active</div>
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("ZhuangTai")%>' prop="status">
					<template slot-scope="scope" class="status-info">
						<div class='successStatus commonStatus' v-if="scope.row.status=='0'"><%=rb.getString("ChengGong") %></div>
						<div class='failStatus commonStatus' v-else-if="scope.row.status=='1'"><%=rb.getString("ShiBai")%></div>
						<div class='runningStatus commonStatus' v-else-if="scope.row.status=='2'"><%=rb.getString("JinXingZhong")%></div>
						<div class='runningStatus commonStatus' v-else-if="scope.row.status=='3'"><%=rb.getString("WeiZhiXing") %></div>
						<div class='skipStatus commonStatus' v-else-if="scope.row.status=='4'"><%=rb.getString("TiaoGuo")%></div>
					</template>
				</el-table-column>
				<el-table-column label='<%=rb.getString("KaiShiShiJian")%>' prop="start_time"></el-table-column>
				<el-table-column label='<%=rb.getString("JieShuShiJian")%>' prop="end_time"></el-table-column>
				<el-table-column label='<%=rb.getString("ShiBaiYuanYin")%>' prop="failure_reason" show-overflow-tooltip="true"></el-table-column>
			</el-ctable>
	 </el-slide>

	 <el-dialog ref="gsmCheckList" :visible.sync="gsmCkDLShow" width='1100px' :append-to-body="true" :close-on-click-modal="false" @close='gsmCkDLShow = false' class="gsmDetectCheckDialog">
		<div slot="title" class='gsmDetectionTitle'>
			<span class="commonText14"><%=rb.getString("GSMSheBeiLieBiao") %></span>
			<span class="commonNotes12"> (<%=rb.getString("SheBeiJianCheTiShi") %>)</span>
		</div>
		<el-form ref='gsmSelectSnListForm' :rules='gsmSelectSnListRules' :model='gsmSelectSnListForm' label-position="top">
			<el-pairgrid :id="'gsmDetectTb'"
				:rownumber="true"
				ref="gsmDetectTb"
				:left-url="gsmDetectURL"
				:height="'400px'"
				row-key="serialNumber"
				:query-params="gsmDetectParams"
				@selection-change='gsmSelectChange'>
				
				<template slot="left">
					<el-table-column type="selection" :reserver-selection="true"></el-table-column>
					<el-table-column width="45" prop="connection_status">
						<template slot-scope="scope">
							<div v-html="connStatusFormatter(scope.row.connection_status,scope.row)"></div>
						</template>
					</el-table-column>
					<el-table-column label='<%=rb.getString("Title_SheBeiBianMa") %>' prop="serialNumber" min-width="120"></el-table-column>
					<el-table-column label='<%=rb.getString("GSMMingCheng") %>' prop="cellName" min-width="100"></el-table-column>
					<el-table-column label='<%=rb.getString("BanBenHao") %>' prop="softwareVersion" min-width="100"></el-table-column>
					<el-table-column label='<%=rb.getString("ChanPinLeiXingBiaoZhi") %>' prop="product" min-width="80"></el-table-column>
					<el-table-column label='<%=rb.getString("SheBeiZu") %>' prop="groupName"min-width="120"></el-table-column>
				</template>
				<template slot="toolbar">
					<div style="position: relative;">
						<div class="searchCon" style="margin-left:18px;">
							<el-input placeholder='<%=rb.getString("Title_SheBeiBianMa")%>' suffic-icon='el-icon-search' v-model="gsmDetectSearchText" style="width:270px;">
								<i slot="suffix" class="el-icon el-icon-common-search searchCls" @click="gsmQueryDetect"></i>
							</el-input>
						</div>
						<div class="editButton" @click="gsmAddBatchSnClick" size="mini">
							<i class="el-icon el-icon-batchInput" style="font-size: 14px;padding-right: 5px;"></i>
							<span><%=rb.getString("PiLiangShuRu")%></span>
						</div>
					</div>
				</template>
				<template slot="right">
					<el-table-column label='<%=rb.getString("Title_SheBeiBianMa") %>' prop="serialNumber" min-width="120"></el-table-column>
					<el-table-column label='<%=rb.getString("GSMMingCheng") %>' prop="cellName" min-width="100"></el-table-column>
				</template>
			</el-pairgrid>
			<el-form-item prop='serialNumber' style="margin: 0;" label-width="0">
				<el-input v-model='gsmSelectSnListForm.serialNumber' v-show="false"></el-input>
			</el-form-item>
		</el-form>
		<div style="padding-top: 30px;">
			<el-button type="primary" @click="gsmSendDetection"><%=rb.getString("QueDing") %></el-button>
			<el-button @click="gsmCkDLShow = false"><%=rb.getString("QuXiao") %></el-button>
		</div>
	</el-dialog>
	<!--batch input-->
	<el-dialog class='dialogStyle' title='<%=rb.getString("TianJia")%>' width='630px' :visible.sync='gsmBatchSnDialog' :append-to-body="true" :close-on-click-modal="false" @close='gsmCloseBatchSn'>
		<el-form ref='gsmBatchSnForm' :rules='gsmBatchSnRules' :model='gsmBatchSnForm' label-position="top">
			<div>
				<label><%=rb.getString("Title_SheBeiBianMa")%></label>
				<el-form-item prop='serialNumber' style="margin-bottom:22px;">
					<el-input v-model='gsmBatchSnForm.serialNumber' type='textarea' :rows="4" style='margin-top:5px;'></el-input>
				</el-form-item>
				<p style='display:flex;color:#BBB'><span class='el-icon el-icon-circle-info' style='font-size:14px;'></span><span style="font-size:12px;"><%=rb.getString("eNBZhuCeTiShiWenZi") %></span></p>
			</div>
			<div style='margin-top:45px;'>
				<el-button @click='gsmSaveBatchSn' type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click='gsmCloseBatchSn'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-form>
	</el-dialog>
</div>

<script>
	var gsmPlugAndPlayVue = new Vue({
		el: '#gsmPlugAndPlayPage',
		data() {
			var gsmValidatorSn = (rule,value,callback) => {
				var serialNumber = value, temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/,
					list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ 
						return item.length > 0;
					});
				
				if (serialNumber == null || serialNumber.length == 0) {
					callback(new Error('<%=rb.getString("SNBuNengWeiKong")%>'));
				}else{
					var nameFlag = list.every(function(item,index){
						return temp.test(item)
					})
					if(nameFlag){
						callback()
					}else{
						callback(new Error('<%=rb.getString("QingShuRuZhengQueSn")%>'));
					}
					
				}
			},
			validatorSNs = (rule,value,callback) => {
				var vm = this;
				
				if(vm.gsmDetectSelections.length == 0) {
					callback('<%=rb.getString("QingXuanZeJiLu")%>');
				}else {
					callback();
				}
			};
			return {
				gsmConfigURL: '${ctx}/gsm/pnp/queryPolicyInfoPageList.action',
				gsmPolicyQuery: {
					searchText: '',
					timeZone: timeZone, 
				},
				gsmPolicyNumber: 0,
				gsmMenus: [],
				gsmStatusURL: '${ctx}/gsm/pnp/getPnPTaskPageList.action',
				gsmStatusQuery: {
					searchText: '',
					productType: '',
					status: '',
					queryStartTime: '',
					queryEndTime: '',
					timeZone: timeZone
				},
				statusSearchText: '',
				gsmSuccessCount: 0,
				gsmFailCount: 0,
				gsmTabsActive: '0',
				productTypeList: [],
				gsmTimeRange: [],
				productModels: [],

				slideHeader: false,
				footerShow: false,
				gsmCurrentRow: [],
				slideTile: '',
                slideURL: '',
                slideUrl:"",
				slideTitle:"",
				slideFooter:"",
				slideHeader:"",
				slidePosition:"",
				slideHeight:"",
				slideWidth:"",
				slideModal:"",

				taskUrl:"",
				params_progress:{
					timeZone:timeZone,
					task_id: '',
					page: 1,
					rows: 50
				},

				gsmDetectURL: '',
				gsmSelectSnListForm:{
					serialNumber:''
				},
				gsmSelectSnListRules:{
					serialNumber:[
						//{validator:validatorSNs,trigger:'change'}
						{required:true,message:'<%=rb.getString("QingXuanZeJiLu")%>',trigger:'change'}
					]
				},
				gsmDetectParams: {
					policyId: '',
					searchText: ''
				},
				gsmCkDLShow: false,
				gsmDetectSelections: [],
				retryLoading: '',
				gsmDetectSearchText: '',
				gsmPolicySwitch: '',
				//batch input sn
				gsmBatchSnDialog: false,
				gsmBatchSnForm:{
					serialNumber: '',
					type: 'input'
				},
				gsmBatchSnRules:{
					serialNumber:[
						{validator: gsmValidatorSn,trigger:'change'}
					]
				},
			}
		},
		watch: {
			gsmTimeRange(newVal){
				var vm = this;

				if(!newVal){
					newVal = [];
					vm.gsmStatusQuery.queryStartTime = '';
	    			vm.gsmStatusQuery.queryEndTime = '';
				}
			},
			gsmDetectSelections(){
				var vm = this,
					data = vm.$refs.gsmDetectTb.getData();
			
				if(data && data.length > 0){
					vm.gsmSelectSnListForm.serialNumber = data.map(function(row){ return row.serialNumber ;}).join(',');
				}else{
					vm.gsmSelectSnListForm.serialNumber = '';
				}
			},
		},
		computed: {
			gsmPnpEnable() {
				return writableMap['CODE_PLUG_AND_PLAY'] == true;
			}
		},
		methods: {
			//gsm query product list
			gsmInit(){
				var vm = this;
				
				axios.post("${ctx}/gsm/pnp/getProductTypeSelect.action").then(function(res){
					var data = res.data,
						arr = [{name:'<%=rb.getString("QuanBu")%>', value:''}];

			   		(data || []).map(function(item){
			   			if (item){
			   				arr.push({name:item.name,value:item.value})
			   			}
			   		})

			   		vm.productTypeList = arr;
				});
			},
			//gsm Add Policy Task
			gsmAddTaskBtnClick() {
				var vm = this,
					rd = Math.random(),
					url = '${ctx}/gsm/pnp/toGSMAddConfig.action?r='+rd;
					
				vm.slideTile = '<%=rb.getString("XinZeng")%> Policy';
				vm.slideURL = url;
				vm.slideHeader = false;
				vm.footerShow = false;
				window.plugAndPlayVue.tabShow = 'itemShow';

				vm.$refs.gsmSlider.showSlide(function(){
					eventBus.$emit('gsmInit-plugPlayConfig','add','');
				});
				
				vm.gsmCancelViewSlide();
			},
			//gsm Add or modify Policy Save
			gsmAddConfig() {
				eventBus.$emit('gsmSave-plugPlayConfig');
			},
			//gsm Add or modify Policy close
			gsmCloseConfig() {
				eventBus.$emit('gsmHandle-plugPlayCancel');
			},
			//refresh  gsm table
			gsmReloadConfig() {
				var vm = this;

				vm.$refs.gsmPolicyList.refresh();
			},
			//gsm gsmRowClick
			gsmRowClick(row) {
				var vm = this;

				vm.gsmCurrentRow = row;
			},
			//gsm Policy list search
			queryPolicyGsm(text) {
				var vm = this;

				vm.gsmPolicyQuery.searchText = text;
			},
			//gsm Execute Status search
			queryGsmStatus(text) {
				var vm= this;

				vm.gsmStatusQuery.searchText = vm.statusSearchText;
			},
			//gsm time search
	    	gsmDateChange(val) {
				var vm = this;

				vm.gsmTimeRange = val;

				if(val != null){
    				vm.gsmStatusQuery.queryStartTime = vm.gsmTimeRange[0];
	    			vm.gsmStatusQuery.queryEndTime = vm.gsmTimeRange[1];
    			}
			},
			//gsm policy list loadSuccess
			gsmPolicyListSuccess(data) {
				var vm = this,
					properties = data.total || 0;

				vm.gsmPolicySwitch = data.rows[0].switch;
				vm.gsmPolicyNumber = properties || 0;
			},
			//gsm Execute Status loadSuccess
			gsmConfigStatusSuccess(data) {
				var vm = this,
					properties = data.properties||{};

				vm.gsmSuccessCount = properties.successCount||0;
				vm.gsmFailCount = properties.failCount||0;
			},
			//gsm Execute Status tabs
			gsmTabsChange(val){
				var vm = this;

				vm.gsmStatusQuery =  {
					searchText: '',
					productType: '',
					status: '',
					queryStartTime: '',
					queryEndTime: '',
					timeZone: timeZone
				};
				vm.gsmTimeRange = [];
				vm.statusSearchText = '';

				if(val == '0'){
					vm.gsmStatusURL= '${ctx}/gsm/pnp/getPnPTaskPageList.action';
				}else {
					vm.gsmStatusURL= '${ctx}/gsm/pnp/getPnPTaskRecordPageList.action';
				}

				if(val == '1'){
					//升级
					vm.gsmStatusQuery.progress = '1';
				}else if(val == '2'){
					//license
					vm.gsmStatusQuery.progress = '2';
				}else if(val == '3'){
					//参数配置
					vm.gsmStatusQuery.progress = '3';
				}

				vm.gsmCancelViewSlide();
			},
			//gsm enable switch click
			gsmSwitchChange(fn) {
				var vm = this, params = {},
					url =  '${ctx}/gsm/pnp/updatePnPPolicySwitch.action';

				vm.$nextTick(function(){
					if(vm.gsmCurrentRow.switch == '0'){
						params.policySwitch= '1';
					}else{
						params.policySwitch= '0';
					}

					params.policyId= vm.gsmCurrentRow.policyId;
					params.policyName = vm.gsmCurrentRow.policyName;

					axios.post(url, stringify(params)).then(function(res){
						var data = res.data;

						if(data.success) {
							fn();
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success'
							});

							vm.$refs.gsmPolicyList.refresh();
						}else {
							vm.$message({
								message: data['message'],
								type: 'error'
							});
						}
					}).catch(function(){})
				})
			},
			//gsm Execute Status: view info
			gsmViewConfigTask(row){
				var vm = this;

				vm.params_progress.task_id = row.task_id;
				vm.taskUrl = "${ctx}/gsm/pnp/getPnPTaskRecordPageList.action";
				vm.$refs.ctableProgress.refresh();
				vm.$refs.slideView.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    });
			},
			//gsm Execute Status: retry
			gsmRestartStatus(row){
				var vm = this, params = {},
					url = '${ctx}/gsm/pnp/updatePnPManualReExecutePolicyTaskStart.action';

				params = {
					taskId: row.task_id 
				};

				vm.retryLoading = row.policyId;

				axios.post(url, stringify(params)).then(function(res){
					var data = res.data;

					if(data['success']) {
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type: 'success'
						});

						vm.$refs.gsmStatusList.refresh();
					}else {
						vm.$message({
							message: data['message'],
							type: 'error'
						});
					}
					vm.retryLoading = '';
				}).catch(function(){});

				vm.gsmCancelViewSlide();
			},
			//gsm Execute status: start
			gsmStartOperStatus(row){
				var vm = this,
					params = {
						taskId: row.task_id
					}

				axios.post("${ctx}/gsm/pnp/updatePnPManualPolicyTaskStart.action",stringify(params)).then(function(response){
					var data = response.data;

					if(data["success"]){
						vm.$refs.gsmStatusList.refresh();
					}else{
						vm.$message.error(data["message"]);
					}
				})

				vm.gsmCancelViewSlide();
			},
			//gsm Execute Status: delete
			gsmDeletStatus(row){
				var vm = this, 
					params = {},
					url = '${ctx}/SON/SelfConfigurationTask/delSelfConfigTask.action';

				params = { taskId: row.task_id };

				vm.$confirm('<%=rb.getString("QueRenShanChu") %>','<%=rb.getString("QueRen") %>').then(function(r){
					if(r) {
						axios.post(url, stringify(params)).then(function(res){
							var data = res.data;

							if(data['success']) {
								vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type: 'success'
								});

								vm.$refs.gsmStatusList.refresh();
							}else {
								vm.$message({
									message: data['message'],
									type: 'error'
								});
							}
						}).catch(function(){});
					}
				}).catch(function(){});

				vm.gsmCancelViewSlide();
			},

			//-------------------------------------------------------------------gsm table operation
			gsmOptClick(row, evt) {
				var vm = this, modifyFlag, delFlag;
				//开关打开时， 不可修改，不可删除
				if(row.switch == '1'){
					modifyFlag = true;
					delFlag = true;
				}else{
					modifyFlag = false;
					delFlag = false;
				}

				vm.gsmMenus= [
					{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info', row: row},
					{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit hidden CODE_PLUG_AND_PLAY",code:'modify', row: row, disable: modifyFlag},
					{label:'<%=rb.getString("JianChe")%>',cls:"el-icon el-icon-operation-scan hidden CODE_PLUG_AND_PLAY",code:'check', row: row},
					{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete hidden CODE_PLUG_AND_PLAY",code:'del', row: row, disable: delFlag}
				];

		    	vm.$nextTick(function(){
		    		document.body.click();
					vm.$refs.gsmMenu.show(evt);
		    	});
			},
			//gsm policy menu click
			gsmMenuClick(row, evt) {
				var vm = this,
					code = row.code,
					data = row.row,
					actions = {
						info: vm.gsmView,
						modify: vm.gsmModify,
						del: vm.gsmDeleteData,
						check: vm.gsmCheckPolicy
					};

				if(actions[code]) {
					actions[code](data);
				}
			},
			//policy info
			gsmView(row) {
				var vm = this,
					rd = Math.random(),
					url = '${ctx}/gsm/pnp/toGSMAddConfig.action?r='+rd,
					params = { policyId: row.policyId };

				vm.slideURL = url;
				vm.slideHeader = false;
				vm.footerShow = false;
				window.plugAndPlayVue.tabShow = 'itemShow';
				vm.$refs.gsmSlider.showSlide(params,function(){
					eventBus.$emit('gsmInit-plugPlayConfig','view', row);
				});

				vm.gsmCancelViewSlide();
			},
			//policy gsmModify
			gsmModify(row){
				var vm = this, 
					rd = Math.random(),
					url = '${ctx}/gsm/pnp/toGSMAddConfig.action?r='+rd,
					params = { policyId: row.policyId };

				vm.slideURL = url;
				vm.slideHeader = false;
				vm.footerShow = false;
				window.plugAndPlayVue.tabShow = 'itemShow';
				vm.$refs.gsmSlider.showSlide(params,function(){
					eventBus.$emit('gsmInit-plugPlayConfig','modify', row);
				});

				vm.gsmCancelViewSlide();
			},
			gsmCheckPolicy(row){
				var vm = this,
					params = {
						policyId: row.policyId
					};

				vm.gsmCkDLShow = true;
				if(vm.$refs.gsmSelectSnListForm){
				    vm.$refs.gsmSelectSnListForm.resetFields();
				}

				Object.assign(vm.gsmDetectParams, params);
				vm.gsmDetectSelections = [];
				vm.gsmDetectParams.searchText = '';
				vm.gsmDetectSearchText = '';
				vm.$nextTick(function(){
					vm.$refs.gsmDetectTb.clear();
					vm.gsmDetectURL = '${ctx}/gsm/pnp/queryPolicyDeviceInfoPageList.action';
				})
			},
			gsmSendDetection() {
				var vm = this,
					params = {
						policyId: vm.gsmCurrentRow.policyId,
						smallCellCode: vm.gsmSelectSnListForm.serialNumber,
					};

				// 下发检测
				vm.$refs.gsmSelectSnListForm.validate((valid) => {
					if(valid){
						axios.post('${ctx}/SON/SelfConfiguration/exeSelfConfigDetectPolicy.action', stringify(params)).then(function(res){
							var data = res.data;
		
							if(data['success']) {
								vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type:'success'
								});
		
								vm.gsmCkDLShow = false;
							}else {
								vm.$message.error(data['message']);
							}
						});
					}
				});
                
			},

			//ENB batch add 
			gsmAddBatchSnClick(){
				var vm = this;

				vm.gsmBatchSnDialog = true;

				if(vm.$refs.gsmBatchSnForm){
					vm.$refs.gsmBatchSnForm.resetFields();
				}
			},
			gsmSaveBatchSn(){
				var vm = this, 
					params = {}, 
					snStr = vm.gsmBatchSnForm.serialNumber,
					list = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});
									
				params.sns = list.join(",");
				params.policyId = vm.gsmCurrentRow.policyId;
				
				vm.$refs.gsmBatchSnForm.validate((valid) => {
					if(valid){
						axios.post('${ctx}/gsm/pnp/querySelectedPolicyDeviceInfos.action', stringify(params)).then((res)=>{
							var data = res.data;
							
							if(data && data.length > 0){		
								vm.$refs.gsmDetectTb.appendCheckedRows(data);

								vm.gsmCloseBatchSn();
							}else{
								vm.$message('<%=rb.getString("MeiYouKePiPeiSheBei")%>')
							}
						})
					}
				})
			},
			gsmCloseBatchSn(){
				var vm = this;

				vm.gsmBatchSnDialog = false;
				vm.$refs.gsmBatchSnForm.resetFields();
			},


			//detect query
			gsmQueryDetect(val){
				var vm = this;

				vm.gsmDetectParams.searchText = vm.gsmDetectSearchText;
			},
			//detece selectChange
			gsmSelectChange(selection) {
				var vm = this;

				vm.gsmDetectSelections = selection;
			},
			//policy delete
			gsmDeleteData(row) {
				var vm = this,
					url =  '${ctx}/gsm/pnp/delPnPPolicy.action',
					params = { 
						policyId: row.policyId,
						policyName: row.policyName
					};

				vm.$confirm('<%=rb.getString("QueRenShanChu") %>','<%=rb.getString("QueRen") %>').then(function(r){
					if(r) {
						axios.post(url,stringify(params)).then(function(res){
							var data = res.data;

							if(data['success']) {
								vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type: 'success'
								});

								vm.$refs.gsmPolicyList.refresh();
							}else {
								vm.$message({
									message: data['message'],
									type: 'error'
								});
							}
						}).catch(function(){});
					}
				}).catch(function(){});
			},
			// table menu hide
			gsmHanderClose() {
				var vm = this;

				vm.$refs.gsmMenu.hide();
			},
			// Execute Status gsmRowClick view
			gsmCancelViewSlide(){
		    	var vm = this;
				
		    	vm.$refs.slideView.hide();
		    },

		    // slide close
		    /*gsmPlugPlayCancelSlide(){
		    	var vm = this;

				vm.$refs.gsmSlider.hide();
			},*/
		},
		mounted() {
			var vm = this; 

			vm.gsmInit();
			eventBus.$off('gsmClose-config').$on('gsmClose-config', vm.gsmCloseConfig);
			eventBus.$off('gsm_reload-config-list').$on('gsm_reload-config-list', vm.gsmReloadConfig);
			//eventBus.$off('gsmCancel-plugPlaySlide').$on('gsmCancel-plugPlaySlide',vm.gsmPlugPlayCancelSlide);

			// 初始化拖拽
			var gsmHLine = document.querySelector('.gsmHorizontalLine');
			gsmPlugAddListener(gsmHLine,"mousedown",onmousedownVGsm);
		}
	});

	// 根据事件对象获取事件触发源对象
	function getTarget(evt){
		return evt.target || evt.srcElement;
	}
	/**
	* 绑定事件方法
	* @param element{dom}: 要绑定事件的对象
	* @param type{string}: 事件类型
	* @param listener{function}: 事件响应的方法
	* @param useCapture{boolean}: 是否在捕获阶段触发
	**/
	function gsmPlugAddListener(element,type,listener,useCapture){
		element.addEventListener?element.addEventListener(type,listener,useCapture):element.attachEvent("on" + type,listener);
	}

	/* 鼠标点击事件 */
	function onmousedownVGsm(event) {
		var _self = this;
		var lastY = event.clientY, d = document,
		rowItems = document.querySelectorAll('.gsmFlexRowItem'),
		preItem = rowItems[0],
		sufItem = rowItems[1];
		_self.clickDown = true;
		d.onmousemove = function(event){
			var evt = event || window.event;
			if(_self.clickDown == true) {
				var direction = evt.clientY - lastY;
				if(direction > 0) {//up
				setHeight(sufItem,direction);
				clearHeight(preItem);
				} else if(direction < 0){//down
				setHeight(preItem,-direction);
				clearHeight(sufItem);
				}
				lastY = evt.clientY;
			}
		};
		d.onmouseup = function(){
			d.onmousemove = null;
			d.onmouseup = null;
			_self.clickDown = false;
			$('#omc_app_ctn').resize();
		};

		function setHeight(item,offset) {
			var style = getComputedStyle(item),
				height = style.height.replace('px','') - offset;
			// maxHeight、minHeight 防止flex布局对高度计算的影响
			if(height < 200) height = 200;
			item.style.height = height + 'px';
			item.style.maxHeight = height + 'px';
			item.style.minHeight = height + 'px';
		}
		function clearHeight(item) {
			item.style.height = '100%';
			item.style.maxHeight = '';
			item.style.minHeight = '';
		}
	}
</script>
