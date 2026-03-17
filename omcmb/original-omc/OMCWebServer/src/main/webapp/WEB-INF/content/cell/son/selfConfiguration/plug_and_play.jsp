<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" language="java" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
	#plugAndPlayPage .resultHeadBox .statisticsSuccessDiv{
		height: 14px;
		line-height: 14px;
		display: flex;
		font-size: 14px;
		border-right: 1px solid #DFE2EE;
	}
	#plugAndPlayPage .statisticsSuccessDiv .el-icon::before{
		color:#67D972;
	}
	#plugAndPlayPage .statisticsSuccessDiv > div:first-child{
		padding: 0px 10px;
		color: #67D972;
	}
	#plugAndPlayPage .statisticsSuccessDiv > div:last-child{
		padding: 0px 10px 0 0;
	}
	#plugAndPlayPage .resultHeadBox .statisticsFailDiv{
		height: 14px;
		line-height: 14px;
		display: flex;
		font-size: 14px;
	}
	#plugAndPlayPage .statisticsFailDiv .el-icon::before{
		color: #E88282;
		content:'\e6fb';
	}
	#plugAndPlayPage .statisticsFailDiv > div:first-child{
		padding: 0px 10px;
		color: #E88282;
	}
	
	i.disabled { opacity: 0.6; }
	#plugAndPlayPage .el-tabs .el-tabs__header { border-bottom: 1px solid #E9E9E9; }
	#plugAndPlayPage .commonHeight { height: 100%; }
	#plugAndPlayPage .commonDisplayBlock { display: block; }
	#plugAndPlayPage .commonBorder { border: 1px solid #E9E9E9; }
	#plugAndPlayPage .commonIconStyle { margin: 10px 15px; width: 30px; height: 30px; color: #7A7992;background: rgba(122,121,146,0.1); border-radius: 100px; line-height: 30px !important; }
	#plugAndPlayPage .commonToolBarBox .el-query .advanceQuery .el-icon { font-size: 14px; }
 
	#plugAndPlayPage .commonToolBarBox .el-query .advanceQuery .el-icon:before,
	#plugAndPlayPage .commonIcon:before,
	#plugAndPlayPage .informationWarp .el-card__header { color: #7A7992; }
	#plugAndPlayPage .commonBorderRadius { border-radius: 10px;}
	#plugAndPlayPage .commonToolBarBox { height: 30px; }
	#plugAndPlayPage .commonLeft20 { padding-left: 20px;}
	#plugAndPlayPage .commonContent { justify-content: space-between; }
	#plugAndPlayPage .addBtn { top: -18px !important; right: -8px;}
	#plugAndPlayPage .eNBTableTitle { line-height: 30px;}
	#plugAndPlayPage .commonToolBarBox .el-query{ right: 56px; }
	#plugAndPlayPage .commonToolBarBox .el-query .advanceQuery { height: 26px; border-radius: 8px; }
	#plugAndPlayPage .commonToolBarBox .el-query .advanceQuery .el-input.el-input--small{ width: 280px; }
	#plugAndPlayPage .commonToolBarBox .el-query .advanceQuery .el-input.el-input--small .el-input__inner { height: 26px; line-height: 26px; }
	#plugAndPlayPage .el-table th { background: #F9F9F9; }
	#plugAndPlayPage .el-table td { color: #666666; }
	#plugAndPlayPage .el-table--border th { border-right: 1px solid #E9E9E9; }
	#plugAndPlayPage .el-table th>.cell { color: #999999; font-weight: normal; }
	#plugAndPlayPage .radioBox .el-radio-button__inner { padding: 0 12px; border: none; line-height: 26px; font-size: 12px; }
	#plugAndPlayPage .radioBox .el-radio-button:first-child .el-radio-button__inner { border-radius: 100px 0 0 100px; border-left: none; }
	#plugAndPlayPage .radioBox .el-radio-button:last-child .el-radio-button__inner { border-radius: 0 100px 100px 0; }
	#plugAndPlayPage .radioBox .el-radio-button__orig-radio:checked+.el-radio-button__inner {
		color:var(--main-color);
		background: rgba(var(--main-color-rgba1),0.1);
		border-color:var(--main-color) !important;
		border-radius: 100px; 
	}
	
	#plugAndPlayPage .radioBox { border: 1px solid #E9E9E9; height: 26px; border-radius: 100px; background: #FFFFFF !important;}
	#plugAndPlayPage .executeStatusQuery .el-query{ right: 0; }
	#plugAndPlayPage .statusSize { font-size: 18px; margin-right: 4px; margin-top: 2px; }
	#plugAndPlayPage .el-icon-status-disable:before { color: #B8C3D9; }
	#plugAndPlayPage .el-icon-status-enable:before { color: #4ED76E; }
	#plugAndPlayPage .statusText { font-size: 12px; color: #666666; }
	#plugAndPlayPage .el-progress-bar { width: 85px; }
	#plugAndPlayPage .el-progress__text { display: none; }
	#plugAndPlayPage .runningStatus {  width: 90px; background: #E4F1FF;  color: #4D84FF; }
	#plugAndPlayPage .successStatus {  width: 76px; background: #EEFFF3;  color: #4ED76E;}
	#plugAndPlayPage .skipStatus { width: 90px; background: #FFFDE3;  color: #FFAA00;}
	#plugAndPlayPage .failStatus { width: 50px; background: #FEF2F2;  color: #FF6D59; }
	#plugAndPlayPage .commonStatus { height: 24px; line-height: 24px; border-radius: 100px; text-align: center; }
	#plugAndPlayPage .commonWidth166 .el-input { width: 150px; }
	#plugAndPlayPage .informationWarp { width: calc(100% - 20px) !important; height: 218px !important; margin: 0 8px; bottom: 50px !important; left: 2px !important; }
	#plugAndPlayPage .informationWarp .el-card { border-radius: 0 0 10px 10px; }
	#plugAndPlayPage .informationWarp .el-card__header span:last-child { right: 4px !important; }
	#plugAndPlayPage .informationWarp .el-card__header .el-icon-close {  font-size: 14px; }
	.plugPlayAddSlide { width: 100% !important; height: 100% !important; border-radius: 10px; }
	.plugPlayAddSlide .el-card__header { height: 40px !important; line-height: 40px !important; }
	#plugAndPlayPage .tabTitleIcon { padding: 0 10px 0 0; }
	#plugAndPlayPage .tabTitleIcon:before { font-size: 16px; }
	#plugAndPlayPage .queryGroup { height: 26px; border-radius: 8px; }
	#plugAndPlayPage .queryGroup .el-input { width: 280px; }
	#plugAndPlayPage .queryGroup .el-input__inner { width: 280px; height: 26px; line-height: 26px; padding: 0; }
	#plugAndPlayPage .queryGroup .el-icon-common-search { font-size: 14px; }
	#plugAndPlayPage .queryGroup .el-icon-common-search:before { color: #7A7992; }
	#plugAndPlayPage .el-date-editor .el-range-input { font-size: 12px; }
	#plugAndPlayPage .el-date-editor .el-range__close-icon { margin-top: -12px; }
	#plugAndPlayPage .specialQuery { margin-left: 10px; }
	#plugAndPlayPage .specialQuery .el-input__inner { width: 200px;}
	#plugAndPlayPage .addBtnStyle { top: 40px; margin-right: 8px !important; }
	#plugAndPlayPage .disabledClass { cursor: not-allowed !important; opacity: 0.4; }
	#plugAndPlayPage .disabledClass:before {color: #c0c4cc;  cursor: not-allowed !important; }
	#plugAndPlayPage .defaultClass { cursor: pointer; }
	
	#plugAndPlayPage .horizontal-line-enb,
	#plugAndPlayPage .horizontal-line-cpe {
		cursor: row-resize;
		padding: 2px;
	}
	#plugAndPlayPage .el-tooltip__popper.is-dark { margin: 0 30px 0 80px; }
	#plugAndPlayPage .el-tabs__item .el-icon::before {
		color:#333;
	}
	#plugAndPlayPage .el-picker-panel {
		margin: 5px -130px;
	}
	#plugAndPlayPage .flex-item-cls {
		height: 100%;
		overflow: auto;   
		position: relative;
		box-sizing: border-box;
	}
	.enbDetectCheckDialog .editButton{
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
	.enbDetectCheckDialog .editButton i{
		font-size:14px !important;
	}
	.enbDetectCheckDialog .editButton span{
		font-size:12px;
	}
	.enbDetectCheckDialog .el-pairgrid-title {
		/*display: none !important;*/
		line-height: 50px !important;
	}
</style> 
<!-- 即插即用 -->
<div id="plugAndPlayPage" class="container" style='position: relative;height: 100%;'>
	
	<!-- eNB,CPE Tab -->
	<el-tabs class='commonHeight' v-model="activeTab" @tab-click="clickTab" :class="tabShow">
		<!-- eNB -->
		<el-tab-pane v-if="activeTab == 'enb' || activeTab == 'gsm'" name="enb">
			<span slot='label'><i class="el-icon el-icon-menu-eNB tabTitleIcon"></i>eNB</span>
			<!-- 操作按钮 -->
			<div class="operations addBtn">
				<div class="circleIcon placeholder-bt addBtnStyle hidden CODE_PLUG_AND_PLAY" placeholder="<%=rb.getString("XinZeng")%>">		
					<span class="el-icon-circle-add el-icon" @click="addTaskBtnClick"></span>
				</div>
			</div>
			<el-ctable ref="enbConfig" class='commonBorderRadius commonBorder flex-row-item-enb' style='margin-top: 10px;margin-bottom: 10px;'
				:url="enbConfigURL" :pagination="false" 
				:query-params="enbQuery"
				row-key="policyId"
				@row-click="rowClick"
				@load-success="enbPolicyListSuccess">
				<div slot="toolbar">
					<div class='commonFlex commonToolBarBox commonContent'>
						<div class='commonTextWeight14 eNBTableTitle commonLeft20'><%=rb.getString("CeLueLieBiao")%>({{enbPolicyNumber}})</div>
						<el-query type="normal" @query="queryEnb" placeholder="<%=rb.getString("CeLueMingChen") %> / <%=rb.getString("MuBiaoBanBen")%>"></el-query>
					</div>
				</div>
				<el-table-column width="40">
					<template slot-scope="scope">
						<div class="el-icon el-icon-main-more" @click="optClick(scope.row, event)" v-clickoutside="handerClose"></div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("ShiFouQiYong")%>" prop="selfStartEnable" width="70" style="position:relative;">
					<template slot-scope="scope">
						<el-switch v-model="scope.row.selfStartEnable" style="height: 18px;"  
							:before-change="switchChange"
							:active-value="'1'" 
							:inactive-value="'0'" 
							active-color="#4D84FF" 
							inactive-color="#CFCFCF">	
						</el-switch>
					</template>
				</el-table-column> 
				<el-table-column label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" prop="productType" show-overflow-tooltip="true" width="240"></el-table-column>
				<el-table-column label="<%=rb.getString("CeLueMingChen") %>" prop="policyName" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label="<%=rb.getString("ZhiXingFangShi")%>" prop="executeType" width="200">
					<template slot-scope="scope">
						<div v-if="scope.row.executeType === '1'"><%=rb.getString("eNBShouDongZhiXing") %></div>
						<div v-if="scope.row.executeType === '0'"><%=rb.getString("eNBZiDongZhiXing") %></div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("RuanJianShengJi")%>" prop="upgradeEnable" show-overflow-tooltip="true">
					<template slot-scope="scope">
						<div v-if='scope.row.upgradeEnable == "0"' class='commonFlex'>
							<span class="el-icon el-icon-status-disable statusSize"></span>
							<span class='statusText'><%=rb.getString("JinYong")%> <span v-show='scope.row.targetVersion !="" && scope.row.targetVersion !=null && scope.row.targetVersion !=undefined'><%=rb.getString("MuBiaoBanBen")%>={{scope.row.targetVersion[0]}}</span></span>
						</div>
		            	<div v-if='scope.row.upgradeEnable == "1"' class='commonFlex'>
							<span class="el-icon el-icon-status-enable statusSize"></span>
							<span class='statusText'><%=rb.getString("QiYong")%> <span v-show='scope.row.targetVersion !="" && scope.row.targetVersion !=null && scope.row.targetVersion !=undefined'><%=rb.getString("MuBiaoBanBen")%>={{scope.row.targetVersion[0]}}</span></span>
						</div>
		          	</template> 
				</el-table-column>
				<el-table-column label="<%=rb.getString("License")%>" prop="licenseEnable" width="200">
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
				<el-table-column label="<%=rb.getString("ENBCanShuZiPeiZhi")%>" prop="selfConfigEnable" width="200">
					<template slot-scope="scope">
						<div v-if='scope.row.selfConfigEnable == "0"' class='commonFlex'>
							<span class="el-icon el-icon-status-disable statusSize"></span>
							<span class='statusText'><%=rb.getString("JinYong")%></span>
						</div>
		            	<div v-if='scope.row.selfConfigEnable == "1"' class='commonFlex'>
							<span class="el-icon el-icon-status-enable statusSize"></span>
							<span class='statusText'><%=rb.getString("QiYong")%></span>
						</div>
		          	</template>
				</el-table-column>
			</el-ctable>
			<div class="horizontal-line-enb"> </div>
			<!-- 执行状态列表-->
			<el-ctable id="enb_config_status_list" height="76%" style="overflow: auto; flex: auto;" ref="enbStatus" class='commonBorderRadius commonBorder flex-row-item-enb'
				:time="6"
				row-key="task_id"
				:url="enbStatusURL"
				:query-params="enbStatusQuery" @selection-change='enbCpeBatchSelect'
				@load-success="enbConfigStatusSuccess">
				<div slot="toolbar">
					<div class='commonFlex' style='margin-bottom: 8px;padding-right: 10px;border-bottom: 1px solid #D5DCEC; position: relative; height: 30px;'>
						<div class='commonTextWeight14 eNBTableTitle commonLeft20' style="line-height: 22px;"><%=rb.getString("ZhiXingZhuangTai")%></div>

						<div class="toolbarHeadBtnBoxCls" style="border-bottom: none; margin: -6px 0 0 0;" v-if="pnpEnable && enbTabsActive == '0'">
							<div class="selectBlukBoxCls">
								<div class="selectMain headBtnItemCls">
									<div class="bulkSelectBtnBoxCls" @click="openBulkSelectTable" style="padding: 0px 0 0 16px; border-right: none;">
										<span class="el-icon-selected el-icon"></span>
										<span class="bulkSelectNumBoxCls">( {{enbDeviceSelection.length}} )</span>
									</div>
									<div class="selectTableBoxCls" style="position: absolute;top: 26px;left: 0px;height: 258px;" v-show="bulkSelectShow">
										<div class="selectBoxTitle">
											<span><%=rb.getString("YiXuan")%></span>
											<span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
										</div>
										<div class="selectBoxMain">
											<div class="tableInfoCls">
												<div class="tableInfoHeader">
													<div><%=rb.getString("Title_SheBeiBianMa")%></div>
													<div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
												</div>
												<el-ctable 
													id="bulkSelectTable" 
													ref="bulkSelectTable" 
													:data="enbDeviceSelection" 
													:showHeader="false"
													:rownumber="false"
													:front-pagination="true"
													height="170px" pagination="true" >
													<el-table-column prop="task_id" v-if="false"></el-table-column>
													<el-table-column width="588">
														<template slot-scope="scope" >
															<div class="tableItemCls">
																<span>{{scope.row.serial_number}}</span>
																<span @click="delBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
															</div>
														</template>
													</el-table-column>
												</el-ctable>
											</div>
										</div>
									</div>
								</div>
							</div>
							<div :class="enbDeviceSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" style="border-right: none;" @click="enbCpeBatchRetryBtn">
								<span class="el-icon el-icon-operation-restart"></span>
								<span><%=rb.getString("ChongXinZhiXing")%></span>
							</div>
						</div>

						<div class="resultHeadBox commonFlex" style="position: absolute; right: 16px; top: 5px;">
							<div class="statisticsSuccessDiv">
								<div><span class="el-icon el-icon-circle-success" style='margin-right: 6px;'></span><%=rb.getString("ChengGong")%></div>
								<div>{{enbSuccessCount}}</div>
							</div>
							<div class="statisticsFailDiv">
								<div><span class="el-icon el-icon-circle-close" style='margin-right: 6px;'></span><%=rb.getString("ShiBai")%></div>
								<div>{{enbFailCount}}</div>
							</div>
						</div>
					</div>
					<div class='commonFlex commonToolBarBox commonContent commonLeft20 executeStatusQuery' style='position: relative;padding-right: 10px;'>
						<el-radio-group v-model='enbTabsActive' class='radioBox' @change="enbTabsChange">
							<el-radio-button label="0"><%=rb.getString("SuoYouRenWu") %></el-radio-button>
							<el-radio-button label="1"><%=rb.getString("RuanJianShengJi") %></el-radio-button>
							<el-radio-button label="2"><%=rb.getString("License") %></el-radio-button>
							<el-radio-button label="3"><%=rb.getString("ENBCanShuZiPeiZhi") %></el-radio-button>
						</el-radio-group>
						<div class='commonFlex'>
							<el-select placeholder='<%=rb.getString("ChanPinLeiXingBiaoZhi") %>' v-model='enbStatusQuery.productType' class='commonWidth166'>
								 <el-option v-for="item in productTypeList" :label="item.name" :value="item.value"></el-option>
							</el-select>
							<el-select placeholder='<%=rb.getString("ZhuangTai") %>' v-model='enbStatusQuery.status' class='commonWidth166' style='margin: 0 10px;'>
								<el-option label="<%=rb.getString("SuoYou")%>" value=" "></el-option>
			                    <el-option label="<%=rb.getString("ChengGong")%>" value="0"></el-option>
			                    <el-option label="<%=rb.getString("ShiBai")%>" value="1"></el-option>
			                    <el-option label="<%=rb.getString("JinXingZhong")%>" value="2"></el-option>
			                    <el-option label="<%=rb.getString("WeiZhiXing")%>" value="3"></el-option>
			                    <el-option label="<%=rb.getString("TiaoGuo")%>" value="4"></el-option>
							</el-select>
							<el-date-picker style='width: 280px'
								v-model="timeRange"
								type="datetimerange"
								value-format="yyyy-MM-dd HH:mm:ss"
								range-separator="一"  
								@change="dateChange"
								start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
								end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
							</el-date-picker>
							<div class='queryGroup specialQuery' style='width: 220px;'>
								<el-input v-model='statusSearchText' @keyup.enter.native="queryEnbStatus" class='pairgrid-query' placeholder='<%=rb.getString("CeLueMingChen") %> / <%=rb.getString("Title_SheBeiBianMa")%>' style='width: 200px;'></el-input>
								<i @click='queryEnbStatus' class="el-icon el-icon-common-search"></i>
							</div>					
						</div>
					</div>
				</div>
				<el-table-column type="selection" :reserve-selection="true" :selectable = "enbIsDisabled" width="50" v-if="pnpEnable && enbTabsActive == '0'" :key='8'></el-table-column>
				<!--status: 0-成功， 1-失败，2-进行中，3-未执行（等待）， 4-跳过，  5-部分成功 已去掉-->
				<!--  手动（1）或未执行（3）的任务可以 start;   删除： 等待，失败，成功，部分成功；      
					重新执行 retry：失败，成功  ;    跳过skip： 可retry ,delete; all: 显示结果页面-->
				<el-table-column width="125" v-if="enbTabsActive == '0'" :key='6'>
					<template slot-scope="scope"> 
						<div class='commonFlex'>
							<div>
								<i v-if="(scope.row.status=='0' || scope.row.status=='1' || scope.row.status=='4') && pnpEnable" @click="restartStatus(scope.row)" :class="[retryLoading==scope.row.policyId  ? 'disabledClass' : 'defaultClass']" class="el-icon el-icon-operation-restart commonIcon defaultClass" title="<%=rb.getString("ChongXinZhiXing")%>"></i> 
								<i v-else class="el-icon el-icon-operation-restart commonIcon disabledClass" title="<%=rb.getString("ChongXinZhiXing")%>"></i> 
							</div>
							<div style="margin-left: 10px;" v-show="scope.row.execute_type === '1'">
								<!-- 只有手动模式显示 -->
								<i v-if="scope.row.status=='3' && pnpEnable" @click="startOperStatus(scope.row)" class="el-icon el-icon-operation-start commonIcon defaultClass" title="<%=rb.getString("ZhiXingNew")%>"></i> 
								<i v-else class="el-icon el-icon-operation-start commonIcon disabledClass" title="<%=rb.getString("ZhiXingNew")%>"></i> 						
							</div>
							<div style="margin-left: 10px;">
								<i v-if="(scope.row.status=='0' || scope.row.status=='1' || scope.row.status=='3' || scope.row.status=='4') && pnpEnable" @click="deletStatus(scope.row)" class="el-icon el-icon-operation-delete commonIcon defaultClass" title="<%=rb.getString("ShanChu")%>"></i> 
								<i v-else class="el-icon el-icon-operation-delete commonIcon disabledClass" title="<%=rb.getString("ShanChu")%>"></i> 
							</div>
							<div style="margin-left: 10px;">
								<i @click="viewConfigTask(scope.row)" class="el-icon el-icon-operation-info commonIcon" title="<%=rb.getString("XinXi")%>"></i>
							</div>
						</div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("Title_SheBeiBianMa")%>" prop="serial_number" width="240" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" prop="product_type" width="160"></el-table-column>
				<el-table-column label="<%=rb.getString("CeLueMingChen") %>" prop="policy_name" width="200" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label="<%=rb.getString("ZhiXingFangShi")%>" prop="execute_type" width="130">
					<template slot-scope="scope">
						<div v-if="scope.row.execute_type === '1'"><%=rb.getString("eNBShouDongZhiXing") %></div>
						<div v-if="scope.row.execute_type === '0'"><%=rb.getString("eNBZiDongZhiXing") %></div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("KaiShiShiJian") %>" prop="start_time" width="150"></el-table-column>
				<el-table-column label="<%=rb.getString("JieShuShiJian") %>" prop="end_time" width="150"></el-table-column>
				<el-table-column label="<%=rb.getString("BuJinDu") %>" prop="execute_procedure" width="210" v-if="enbTabsActive == '0'" key='execute_procedure'></el-table-column>
				<el-table-column label="<%=rb.getString("ChuShiBanBen") %>" prop="original_version" width="150" key='original_version' v-if="enbTabsActive == '1'"></el-table-column>
				<el-table-column label="<%=rb.getString("MuBiaoBanBen") %>" prop="target_version" width="150" key='target_version' v-if="enbTabsActive == '1'"></el-table-column>
				<el-table-column label="<%=rb.getString("LicenseWenJian") %>" prop="lic_file" width="200" key='lic_file' v-if="enbTabsActive == '2'"></el-table-column>
				<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="status" width="110">					
					<template slot-scope="scope" class="status-info">
						<div class='successStatus commonStatus' v-if="scope.row.status=='0'"><%=rb.getString("ChengGong") %></div>
						<div class='failStatus commonStatus' v-if="scope.row.status=='1'"><%=rb.getString("ShiBai")%></div>
						<div class='runningStatus commonStatus' v-if="scope.row.status=='2'"><%=rb.getString("JinXingZhong")%></div>
						<div class='runningStatus commonStatus' v-if="scope.row.status=='3'"><%=rb.getString("WeiZhiXing") %></div>
						<div class='skipStatus commonStatus' v-if="scope.row.status=='4'"><%=rb.getString("TiaoGuo")%></div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("ShiBaiYuanYin")%>" prop="failure_reason" show-overflow-tooltip="true" min-width="200"></el-table-column>
			</el-ctable>
			<el-cmenu ref="enbMenu" :data="enbMenus" @click="enbMenuClick"></el-cmenu>
		</el-tab-pane>
		<el-tab-pane v-if="activeTab != 'cpe' && gsmEnable == 'true'" label='GSM' name="gsm">
			<div id="gsmPlayPage" class="flex-item-cls" style="min-width: 1060px;"></div>
		</el-tab-pane>
		<!-- CPE :url="cpeConfigURL"-->
		<el-tab-pane v-if="activeTab == 'cpe'" name="cpe">
			<span slot='label'><i class="el-icon el-icon-menu-CPE tabTitleIcon"></i>CPE</span>
			<!-- 操作按钮 -->
			<div class="operations addBtn">
				<div class="circleIcon placeholder-bt addBtnStyle hidden CODE_PLUG_AND_PLAY" placeholder="<%=rb.getString("XinZeng")%>">		
					<span class="el-icon-circle-add el-icon" @click="addTaskBtnClick"></span>
				</div>
			</div>
			<el-ctable ref="cpeConfig" style="margin-top: 10px; overflow: auto; flex: auto;margin-bottom: 10px;" class='commonBorderRadius commonBorder flex-row-item-cpe'
				row-key="policyId" :pagination="false" 
				:url="cpeConfigURL"
				:query-params="cpeQuery"
				@row-click="rowClick"
				@load-success="cpePolicyListSuccess">
				<div slot="toolbar">
					<div class='commonFlex commonToolBarBox commonContent'>
						<div class='commonTextWeight14 eNBTableTitle commonLeft20'><%=rb.getString("CeLueLieBiao")%>({{cpePolicyNumber}})</div>
						<el-query type="normal" @query="queryCpe" placeholder="<%=rb.getString("CeLueMingChen") %> / <%=rb.getString("MuBiaoBanBen")%>"></el-query>
					</div>
				</div>
				<el-table-column width="40">
					<template slot-scope="scope">
						<div class="el-icon el-icon-main-more" @click="optClick(scope.row, event)" v-clickoutside="handerClose"></div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("ShiFouQiYong")%>" prop="policySwitch" width="70" style="position:relative;">
					<template slot-scope="scope">
							<el-switch :disabled="scope.row.operatorEnable==false" size="mini" v-model="scope.row.policySwitch" :before-change="switchChange"></el-switch>
					</template>
				</el-table-column>				
				<el-table-column label="<%=rb.getString("ChanPinXingHao")%>" prop="productType"></el-table-column>
				<el-table-column label="<%=rb.getString("CeLueMingChen") %>" prop="policyName" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label="<%=rb.getString("RuanJianShengJi")%>" prop="upgradeEnable" show-overflow-tooltip="true">
					<template slot-scope="scope">
						<div v-if='scope.row.upgradeEnable == false' class='commonFlex'>
							<span class="el-icon el-icon-status-disable statusSize"></span>
							<span class='statusText'><%=rb.getString("JinYong")%> <span v-show='scope.row.targetVersion !="" && scope.row.targetVersion !=null && scope.row.targetVersion !=undefined && scope.row.targetVersion.split("=")[1] != "null"'>{{scope.row.targetVersion}}</span></span>
						</div>
		            	<div v-if='scope.row.upgradeEnable == true' class='commonFlex'>
							<span class="el-icon el-icon-status-enable statusSize"></span>
							<span class='statusText'><%=rb.getString("QiYong")%> <span v-show='scope.row.targetVersion !="" && scope.row.targetVersion !=null && scope.row.targetVersion !=undefined && scope.row.targetVersion.split("=")[1] != "null"'>{{scope.row.targetVersion}}</span></span>
						</div>
		          	</template> 
				</el-table-column> 
				<el-table-column label="<%=rb.getString("ENBCanShuZiPeiZhi")%>" prop="configEnable">
					<template slot-scope="scope">
						<div v-if='scope.row.configEnable == false' class='commonFlex'>
							<span class="el-icon el-icon-status-disable statusSize"></span>
							<span class='statusText'><%=rb.getString("JinYong")%></span>
						</div>
		            	<div v-if='scope.row.configEnable == true' class='commonFlex'>
							<span class="el-icon el-icon-status-enable statusSize"></span>
							<span class='statusText'><%=rb.getString("QiYong")%></span>
						</div>
		          	</template>
				</el-table-column>
			</el-ctable>
			<div class="horizontal-line-cpe"> </div>
			<!--CPE 执行状态列表 -->
			<el-ctable id="cpe_config_status_list" height="76%" ref="cpeStatus" style="overflow: auto; flex: auto;" class='commonBorderRadius commonBorder flex-row-item-cpe'
				:time="6"
				row-key="id"
				:url="cpeStatusURL"
				:query-params="cpeStatusQuery" @selection-change='enbCpeBatchSelect'
				@load-success="cpeConfigStatusSuccess">
				<div slot="toolbar">
					<div class='commonFlex' style='margin-bottom: 8px;padding-right: 10px;border-bottom: 1px solid #D5DCEC; position: relative;height: 30px;'>
						<div class='commonTextWeight14 eNBTableTitle commonLeft20' style="line-height: 22px;"><%=rb.getString("ZhiXingZhuangTai")%></div>

						<div class="toolbarHeadBtnBoxCls" style="border-bottom: none; margin: -6px 0 0 0;" v-if="pnpEnable && cpeTabsActive == '0'">
							<div class="selectBlukBoxCls">
								<div class="selectMain headBtnItemCls">
									<div class="bulkSelectBtnBoxCls" @click="openBulkSelectTable" style="padding: 0px 0 0 16px; border-right: none;">
										<span class="el-icon-selected el-icon"></span>
										<span class="bulkSelectNumBoxCls">( {{enbDeviceSelection.length}} )</span>
									</div>
									<div class="selectTableBoxCls" style="position: absolute;top: 26px;left: 0px;height: 258px;" v-show="bulkSelectShow">
										<div class="selectBoxTitle">
											<span><%=rb.getString("YiXuan")%></span>
											<span style="position:absolute;right:20px;top:15px;" class="el-icon el-icon-close" @click="closeBulkSelectTable"></span>
										</div>
										<div class="selectBoxMain">
											<div class="tableInfoCls">
												<div class="tableInfoHeader">
													<div><%=rb.getString("Title_SheBeiBianMa")%></div>
													<div @click="clearBulkSelected"><span style="margin-right:5px;" class="el-icon el-icon-operation-delete" ></span>Clear</div>
												</div>
												<el-ctable 
													id="bulkSelectTable" 
													ref="bulkSelectTable" 
													:data="enbDeviceSelection" 
													:showHeader="false"
													:rownumber="false"
													:front-pagination="true"
													height="170px" pagination="true" >
													<el-table-column prop="id" v-if="false"></el-table-column>
													<el-table-column width="588">
														<template slot-scope="scope" >
															<div class="tableItemCls">
																<span>{{scope.row.serialNumber}}</span>
																<span @click="delBulkSelected(scope.row)" class="el-icon el-icon-circle-close item_show"></span>
															</div>
														</template>
													</el-table-column>
												</el-ctable>
											</div>
										</div>
									</div>
								</div>
							</div>
							<div :class="enbDeviceSelection.length >0 ? 'headBtnItemCls' : 'headBtnItemCls headBtnItemDisCls'" style="border-right: none;" @click="enbCpeBatchRetryBtn">
								<span class="el-icon el-icon-operation-restart"></span>
								<span><%=rb.getString("ChongXinZhiXing")%></span>
							</div>
						</div>

						<div class="resultHeadBox commonFlex" style="position: absolute; right: 16px; top: 5px;">
							<div class="statisticsSuccessDiv">
								<div><span class="el-icon el-icon-circle-success" style='margin-right: 6px;'></span><%=rb.getString("ChengGong")%></div>
								<div>{{cpeSuccessCount}}</div>
							</div>
							<div class="statisticsFailDiv">
								<div><span class="el-icon el-icon-circle-close" style='margin-right: 6px;'></span><%=rb.getString("ShiBai")%></div>
								<div>{{cpeFailCount}}</div>
							</div>
						</div>
					</div>
					<div class='commonFlex commonToolBarBox commonContent commonLeft20 executeStatusQuery' style='position: relative;padding-right: 10px;'>
						<el-radio-group v-model='cpeTabsActive' class='radioBox' @change="cpeTabsChange">
							<el-radio-button label="0"><%=rb.getString("SuoYouRenWu") %></el-radio-button>
							<el-radio-button label="1"><%=rb.getString("RuanJianShengJi") %></el-radio-button>
							<el-radio-button label="3"><%=rb.getString("ENBCanShuZiPeiZhi") %></el-radio-button>
						</el-radio-group>
						<div class='commonFlex'>
							<el-select placeholder='<%=rb.getString("ChanPinXingHao") %>' v-model='cpeStatusQuery.productType' class='commonWidth166'>
								<el-option v-for="item in productModels" :key="item.value" :label="item.label" :value="item.value"></el-option>
							</el-select>
							<el-select placeholder='<%=rb.getString("ZhuangTai") %>' v-model='cpeStatusQuery.status' class='commonWidth166' style='margin: 0 10px;'>
								<el-option label="<%=rb.getString("SuoYou")%>" value=" "></el-option>
			                    <el-option label="<%=rb.getString("ChengGong")%>" value="success"></el-option>
			                    <el-option label="<%=rb.getString("ShiBai")%>" value="fail"></el-option>
			                    <el-option label="<%=rb.getString("JinXingZhong")%>" value="running"></el-option>
			                    <el-option label="<%=rb.getString("DengDai")%>" value="waitting"></el-option>
			                    <el-option label="<%=rb.getString("TiaoGuo")%>" value="skipped"></el-option>
							</el-select>
							<el-date-picker style='width:280px'
								v-model="cpeTimeRange"
								type="datetimerange"
								value-format="yyyy-MM-dd HH:mm:ss"
								range-separator="一"  
								@change="cpeDateChange"
								start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
								end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
							</el-date-picker> 
							<div class='queryGroup specialQuery' style='width: 220px;'>
								<el-input v-model='cpeStatusSearchText' @keyup.enter.native="queryCpeStatus" class='pairgrid-query' placeholder='<%=rb.getString("CeLueMingChen") %> / <%=rb.getString("Title_SheBeiBianMa")%>' style='width: 200px;'></el-input>
								<i @click='queryCpeStatus' class="el-icon el-icon-common-search"></i>
							</div>
						</div>
					</div>
				</div>
				<el-table-column type="selection" :reserve-selection="true" :selectable = "cpeIsDisabled" width="50" v-if="pnpEnable && cpeTabsActive == '0'" :key='88'></el-table-column>
				<el-table-column width="90" v-if="cpeTabsActive == '0'" :key='66'>
					<template slot-scope="scope">
						<div style='display: flex;'>
							<div v-if="scope.row.status=='running' || !pnpEnable">
								<i class="el-icon el-icon-operation-restart disabled commonIcon"></i> 
								<i class="el-icon el-icon-operation-delete disabled commonIcon" style="margin-left: 5px;"></i>							
							</div>
							<div v-if="scope.row.status!='running' && pnpEnable">
								<i class="el-icon el-icon-operation-restart commonIcon" title="<%=rb.getString("ChongXinZhiXing")%>" @click="restartStatus(scope.row)" :class="[retryLoading==scope.row.policyId  ? 'disabledClass' : 'defaultClass']"></i> 
								<i class="el-icon el-icon-operation-delete commonIcon" @click="deletStatus(scope.row)" style="margin-left: 5px;"></i>
							</div>
							<i @click="cpeViewConfigTask(scope.row)" class="el-icon el-icon-operation-info commonIcon" title="<%=rb.getString("XinXi")%>" style="margin-left: 5px;"></i>
						</div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("Title_SheBeiBianMa")%>" prop="serialNumber" width="240" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label="<%=rb.getString("ChanPinXingHao")%>" prop="productType" width="160"></el-table-column>
				<el-table-column label="<%=rb.getString("CeLueMingChen") %>" prop="policyName" width="200" show-overflow-tooltip="true"></el-table-column>			
				<el-table-column label="<%=rb.getString("KaiShiShiJian") %>" prop="startTime" width="150"></el-table-column>
				<el-table-column label="<%=rb.getString("JieShuShiJian") %>" prop="endTime" width="150"></el-table-column>
				<el-table-column label="<%=rb.getString("BuJinDu") %>" prop="step" width="210" v-if="cpeTabsActive == '0'" key='step'></el-table-column>
				<el-table-column label="<%=rb.getString("ChuShiBanBen") %>" prop="original_version" width="150" key='original_version' v-if="cpeTabsActive == '1'"></el-table-column>
				<el-table-column label="<%=rb.getString("MuBiaoBanBen") %>" prop="target_version" width="150" key='target_version' v-if="cpeTabsActive == '1'"></el-table-column>
				<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="status" width="110">
					<template slot-scope="scope" class="status-info">
						<div class='successStatus commonStatus' v-if="scope.row.status=='success'"><%=rb.getString("ChengGong") %></div>
						<div class='failStatus commonStatus' v-if="scope.row.status=='fail'"><%=rb.getString("ShiBai")%></div>
						<div class='runningStatus commonStatus' v-if="scope.row.status=='running'"><%=rb.getString("JinXingZhong")%></div>
						<div class='runningStatus commonStatus' v-if="scope.row.status=='waitting'"><%=rb.getString("DengDai")%></div>
						<div class='skipStatus commonStatus' v-if="scope.row.status=='skipped'"><%=rb.getString("TiaoGuo")%></div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("ShiBaiYuanYin")%>" prop="failureReason" show-overflow-tooltip="true" min-width="200"></el-table-column>				
			</el-ctable>

			<el-cmenu ref="cpeMenu" :data="cpeMenus" @click="cpeMenuClick"></el-cmenu>
		</el-tab-pane>
	</el-tabs>
	<!-- Add Config Slider -->
	<el-slide ref="slider" title="<%=rb.getString("XinZeng")%>" 
		:url="slideURL" 
		:title="slideTile" 
		:header='slideHeader' 
		:footer="footerShow" class='plugPlayAddSlide'
		@ok="addConfig"
		@cancel="closeConfig">
	</el-slide>
	
	<!-- enb execute status 详情 -->
	<el-slide ref="slideView" title="Devices Execute Information" :footer="false" position="bottom" class='informationWarp' 
	    height="300px" :modal='modal' @cancel='cancelViewSlide'>
	    	<el-ctable id="exeProgressTable" ref="ctableProgress" :url="taskUrl" :query-params="params_progress" time="6"
				:row-key="'id'" :pagination="false" :rownumber="false">
				<el-table-column label="<%=rb.getString("JinDu")%>" prop="progress">
					<template slot-scope="scope" class="status-info">
						<div v-if="scope.row.progress=='1'"><%=rb.getString("RuanJianShengJi")%></div>
						<div v-else-if="scope.row.progress=='2'"><%=rb.getString("License")%></div>
						<div v-else-if="scope.row.progress=='3'"><%=rb.getString("ENBCanShuZiPeiZhi") %></div>
						<div v-else-if="scope.row.progress=='4'">Cell active</div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="status">
					<template slot-scope="scope" class="status-info">
						<div class='successStatus commonStatus' v-if="scope.row.status=='0'"><%=rb.getString("ChengGong") %></div>
						<div class='failStatus commonStatus' v-else-if="scope.row.status=='1'"><%=rb.getString("ShiBai")%></div>
						<div class='runningStatus commonStatus' v-else-if="scope.row.status=='2'"><%=rb.getString("JinXingZhong")%></div>
						<div class='runningStatus commonStatus' v-else-if="scope.row.status=='3'"><%=rb.getString("WeiZhiXing") %></div>
						<div class='skipStatus commonStatus' v-else-if="scope.row.status=='4'"><%=rb.getString("TiaoGuo")%></div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("KaiShiShiJian")%>" prop="start_time"></el-table-column>
				<el-table-column label="<%=rb.getString("JieShuShiJian")%>" prop="end_time"></el-table-column>
				<el-table-column label="<%=rb.getString("ShiBaiYuanYin")%>" prop="failure_reason" show-overflow-tooltip="true"></el-table-column>
			</el-ctable>
	 </el-slide>
	 <!-- cpe execute status 详情 -->
	<el-slide ref="cpeSlideView" title="Devices Execute Information" :footer="false" position="bottom" class='informationWarp' 
	    height="300px" :modal='modal' @cancel='cpeCancelViewSlide'>
	    	<el-ctable id="cpeExeProgressTable" ref="cpeCtableProgress" :url="cpeTaskUrl" :query-params="cpe_params_progress" time="6"
				:row-key="'id'" :pagination="false" :rownumber="false">
				<el-table-column label="<%=rb.getString("JinDu")%>" prop="taskType">
					<template slot-scope="scope" class="status-info">
						<div v-if="scope.row.taskType=='upgrade'"><%=rb.getString("RuanJianShengJi")%></div>
						<div v-else><%=rb.getString("ENBCanShuZiPeiZhi") %></div>
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("ZhuangTai")%>" prop="status">
					<template slot-scope="scope" class="status-info">
						<div class='successStatus commonStatus' v-if="scope.row.status=='success'"><%=rb.getString("ChengGong") %></div>
						<div class='failStatus commonStatus' v-else-if="scope.row.status=='fail'"><%=rb.getString("ShiBai")%></div>
						<div class='runningStatus commonStatus' v-else-if="scope.row.status=='running'"><%=rb.getString("JinXingZhong")%></div>
						<div class='runningStatus commonStatus' v-else-if="scope.row.status=='waitting'"><%=rb.getString("DengDai")%></div>	
						<div class='skipStatus commonStatus' v-else-if="scope.row.status=='skipped'"><%=rb.getString("TiaoGuo")%></div>				
					</template>
				</el-table-column>
				<el-table-column label="<%=rb.getString("KaiShiShiJian")%>" prop="startTime"></el-table-column>
				<el-table-column label="<%=rb.getString("JieShuShiJian")%>" prop="endTime"></el-table-column>
				<el-table-column label="<%=rb.getString("ShiBaiYuanYin")%>" prop="failureReason" show-overflow-tooltip="true"></el-table-column>
			</el-ctable>
	</el-slide>
	 
	<el-dialog ref="checkList" :visible.sync="ckDLShow" width='1100px' :append-to-body="true" :close-on-click-modal="false" @close='ckDLShow = false' class="enbDetectCheckDialog">
		<div slot="title" style="padding: 15px 0;display: flex;align-items: center;">
			<span class="commonText14"><%=rb.getString("JiZhanSheBeiLieBiao") %></span>
			<span class="commonNotes12"> (<%=rb.getString("SheBeiJianCheTiShi") %>)</span>
		</div>
		<el-form ref='addListForm' :rules='addListRules' :model='addListForm' label-position="top">
			<el-pairgrid :id="'detectTb'"
				:rownumber="true"
				ref="detectTb"
				:left-url="detectURL"
				:height="'400px'"
				row-key="serialNumber"
				:query-params="detectParams"
				@selection-change='selectChange'>
				
				<template slot="left">
					<el-table-column type="selection" :reserver-selection="true"></el-table-column>
					<el-table-column width="45" prop="connection_status">
						<template slot-scope="scope">
							<div v-html="connStatusFormatter(scope.row.connection_status,scope.row)"></div>
						</template>
					</el-table-column>
					
					<el-table-column label="<%=rb.getString("Title_SheBeiBianMa") %>" prop="serialNumber" min-width="120"></el-table-column>
					<!-- enb 列 -->
					<el-table-column label="<%=rb.getString("HostName") %>" prop="cellName" min-width="100"></el-table-column>
					<el-table-column label="<%=rb.getString("BanBenHao") %>" prop="softwareVersion" min-width="100"></el-table-column>
					<el-table-column label="<%=rb.getString("ChanPinLeiXingBiaoZhi") %>" prop="product" min-width="80"></el-table-column>
					<el-table-column label="<%=rb.getString("SheBeiZu") %>" prop="groupName" min-width="120"></el-table-column>
				</template>
				<template slot="toolbar">
					<div style="position: relative;">
						<div class="searchCon" style="margin-left:18px;">
							<el-input placeholder="<%=rb.getString("Title_SheBeiBianMa")%>" suffic-icon='el-icon-search' v-model="detectSearchText" style="width:270px;">
								<i slot="suffix" class="el-icon el-icon-common-search" style='margin-top: 5px; font-size: 16px;' @click="queryDetect"></i>
							</el-input>
						</div>
						<div class="editButton" @click="enbAddBatchSnClick" size="mini">
							<i class="el-icon el-icon-batchInput" style="font-size: 14px;padding-right: 5px;"></i>
							<span><%=rb.getString("PiLiangShuRu")%></span>
						</div>
					</div>
				</template>
				<template slot="right">
					<el-table-column label="<%=rb.getString("Title_SheBeiBianMa") %>" prop="serialNumber" min-width="120"></el-table-column>
					<el-table-column label="<%=rb.getString("HostName") %>" prop="cellName" min-width="100"></el-table-column>
				</template>
			</el-pairgrid>
			<el-form-item prop='serialNumber' style="margin: 0;" label-width="0">
				<el-input v-model='addListForm.serialNumber' v-show="false"></el-input>
			</el-form-item>
		</el-form>
		<div style="padding: 30px 0 0 0;">
			<el-button type="primary" @click="sendDetection"><%=rb.getString("QueDing") %></el-button>
			<el-button @click="ckDLShow = false"><%=rb.getString("QuXiao") %></el-button>
		</div>
	</el-dialog>
	<!--batch input-->
	<el-dialog class='dialogStyle' title='<%=rb.getString("TianJia")%>' width='630px' :visible.sync='batchSnDialog' :append-to-body="true" :close-on-click-modal="false" @close='closeBatchSn'>
		<el-form ref='batchSnForm' :rules='batchSnRules' :model='batchSnForm' label-position="top">
			<div>
				<label><%=rb.getString("Title_SheBeiBianMa")%></label>
				<el-form-item prop='serialNumber' style="margin-bottom:22px;">
					<el-input v-model='batchSnForm.serialNumber' type='textarea' :rows="4" style='margin-top:5px;'></el-input>
				</el-form-item>
				<p style='display:flex;color:#BBB'><span class='el-icon el-icon-circle-info' style='font-size:14px;'></span><span style="font-size:12px;"><%=rb.getString("eNBZhuCeTiShiWenZi") %></span></p>
			</div>
			<div style='margin-top:45px;'>
				<el-button @click='saveBatchSn' type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click='closeBatchSn'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-form>
	</el-dialog>

	<!-- retry-->
	<el-dialog title='<%=rb.getString("QueRen")%>' width="600px" :visible="enbCpeShoBatchRetryShow" :close-on-click-modal="false" :modal-append-to-body="false" @close="closeEnbCpeBatchRetry">
		<el-form label-position="top" ref="enbCpeBatchRetryForm" :model='enbCpeBatchRetryForm'>
         	<div class="commonSize14"><%=rb.getString("ChongShiRenWu")%></div>
         	<div class="commonNotes12" style="padding: 10px 0;"><%=rb.getString("WuFaChongShiTiShi")%></div>
			<el-checkbox v-model="enbCpeBatchRetryForm.isContainSuccessTask" :true-label="1" :false-label="0" style="line-height: 14px;"><%=rb.getString("TongShiZhiXingChengGongRenWu")%></el-checkbox>
        </el-form>
       	<div slot="footer">
          	 <el-button type="primary" @click="enbCpeBatchRetryConfirm"><%=rb.getString("QueDing")%></el-button>
             <el-button @click="closeEnbCpeBatchRetry"><%=rb.getString("QuXiao")%></el-button>
        </div>			
	</el-dialog>
</div>

<script>
if(window.plugAndPlayVue) {
	try {
		window.plugAndPlayVue.$destroy();
	}catch(e){}
}
window.plugAndPlayVue = new Vue({
		el: '#plugAndPlayPage',
		data() {
			var netType = sysMain.headType,
				validatorSn = (rule,value,callback) => {
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
					
					if(vm.detectSelections.length == 0) {
						callback('<%=rb.getString("QingXuanZeJiLu")%>');
					}else {
						callback();
					}
				};

			return {
				activeTab: ['enb','cpe'].includes(netType)? netType : 'enb',
				enbConfigURL: '${ctx}/SON/SelfConfiguration/querySelfConfigPolicyPageList.action',
				enbQuery: {
					searchText: '',
					timeZone: timeZone,
					page: 1,
					rows: 50
				},
				enbPolicyNumber: 0,
				enbMenus: [],				
				enbStatusURL: '${ctx}/SON/SelfConfigurationTask/getSelfConfigTaskPageList.action',
				enbStatusQuery: {
					searchText: '',
					productType: '',
					status: '',
					queryStartTime: '',
					queryEndTime: '',
					timeZone: timeZone
				},
				statusSearchText: '',
				enbSuccessCount: 0,
				enbFailCount: 0,
				enbTabsActive: '0',
				productTypeList: [],
				timeRange: [],
				//////////////CPE
				cpeConfigURL: '${ctx}/plugandplay/policy/queryAutoPolicyInfoCPEPageList.action',
				cpeQuery: {
					searchText: '',
					timeZone: timeZone,
					page: 1,
					rows: 50
				},
				cpePolicyNumber: 0,
				cpeMenus: [],
				cpeStatusURL: '${ctx}/plugandplay/task/queryAutoPolicyInfoCPEPageList.action',
				cpeStatusQuery: {
					searchText: '',
					productType: '',
					status: '',
					queryStartTime: '',
					queryEndTime: '',
					timeZone: timeZone
				},
				cpeStatusSearchText: '',
				cpeSuccessCount: 0,
				cpeFailCount: 0,
				cpeTabsActive: '0',
				cpeTimeRange: [],				
				productModels: [],		
				slideHeader: false,
				footerShow: false,
				currentRow: [],
				productCpeList: [],
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
				taskUrl:"${ctx}/SON/SelfConfigurationTask/getSelfConfigTaskRecordPageList.action",
				params_progress:{
					timeZone:timeZone,
					task_id: '',
					page: 1,
					rows: 50
				},
				cpeTaskUrl: '${ctx}/plugandplay/task/queryAutoPolicyTaskRecordPageList.action',
				cpe_params_progress:{
					timeZone: timeZone,
					taskId: '',
					searchText: '',
					page: 1,
					rows: 50
				},
				//detectURL: '${ctx}/pm/template/getEnbListPageData.action?isGnb=1',
				detectURL: '',
				addListForm:{
					serialNumber:''
				},
				addListRules:{
					serialNumber:[
						//{validator:validatorSNs,trigger:'change'}
						{required:true,message:'<%=rb.getString("QingXuanZeJiLu")%>',trigger:'change'}
					]
				},
				detectData: [
					{
						serialNumber: '123456',
						product: 'QAFA'
					},
					{
						serialNumber: '12345677',
						product: 'QAFA'
					},
					{
						serialNumber: '12345688',
						product: 'QAFA'
					},
				],
				detectParams: {
					policyId: '',
					searchText: ''
				},
				ckDLShow: false,
				detectSelections: [],
				retryLoading: '',
				detectSearchText: '',
				bulkSelectShow: false,
				enbDeviceSelection: [],
				//retry 确认
				enbCpeShoBatchRetryShow:false,
				enbCpeBatchRetryForm: {
					isContainSuccessTask: '0'
				},
				tabShow: '',
				gsmEnable: '${gsmEnable}',

				batchSnDialog: false,
				batchSnForm:{
					serialNumber: '',
					type: 'input'
				},
				batchSnRules:{
					serialNumber:[
						{validator: validatorSn,trigger:'change'}
					]
				},
			}
		},
		watch: {
			timeRange(newVal){
				var vm = this;
				if(!newVal){
					newVal = [];
					vm.enbStatusQuery.queryStartTime = '';
	    			vm.enbStatusQuery.queryEndTime = '';
				}
			},
			cpeTimeRange(newVal){
				var vm = this;
				if(!newVal){
					newVal = [];
					vm.cpeStatusQuery.queryStartTime = '';
	    			vm.cpeStatusQuery.queryEndTime = '';
				}
			},
			netType(type) {
				var vm = this,
					types = ['enb','cpe'];
	
				if(types.includes(type)) vm.activeTab = type;
				
				vm.enbDeviceSelection = [];
				//清空enb 和 cpe 状态列表的已选数据
				vm.$nextTick(function(){

					if(vm.activeTab == 'enb'){
						vm.$refs["enbStatus"].clearSelection();	
					}else{
						vm.$refs["cpeStatus"].clearSelection();
					}
				})
                vm.bulkSelectShow = false;

				vm.cancelViewSlide();
				vm.cpeCancelViewSlide();
			},
			detectSelections(){
				var vm = this,
					data = vm.$refs.detectTb.getData();
			
				if(data && data.length > 0){
					vm.addListForm.serialNumber = data.map(function(row){ return row.serialNumber ;}).join(',');
				}else{
					vm.addListForm.serialNumber = '';
				}
			},
		},
		computed: {
			isAdmin() {
				return is_super_user == 'true';
			},
			
			pnpEnable() {

				return writableMap['CODE_PLUG_AND_PLAY'] == true;
			},
			netType() {
				var vm = this,
					neType = sysMain.headType;
	
				return neType;
			}
		},
		methods: {		
			init(){
				var vm = this;
				//eNB获取产品类型
				axios.post("${ctx}/SON/SelfConfiguration/getProductTypeSelect.action",stringify({
                	type: 'all'
                })).then(function(res){
					var data = res.data;
			   		
			   		var arr = [{name:'<%=rb.getString("QuanBu")%>',value:''}];
			   		(data || []).map(function(item){
			   			if (item){
			   				arr.push({name:item.name,value:item.value})
			   			}
			   		})
			   		vm.productTypeList = arr.filter(function(item,index){
						return item.value != 'CR-B4860/EU' && item.value != 'CR-B4860/RU'
					})
				});
				//CPE获取产品型号
                axios.post("${ctx}/plugandplay/policy/getProductModelList.action").then(function(res){
	                var data = res.data;
			   		var arr = [{label:'<%=rb.getString("QuanBu")%>',value:''}];
			   		(data || []).map(function(item){
			   			if (item){
			   				arr.push({label:item,value:item})
			   			}
			   		})
			   		vm.productModels = arr;
				}).catch(function(){});	
			},
			clickTab(){
				var vm = this;

				if(vm.activeTab == 'gsm'){
					$('#gsmPlayPage').addClass('loading');
					$("#gsmPlayPage").load('${ctx}/gsm/pnp/toGSMPnP.action',function(data){
						$.parser.parse(this);
						$('#gsmPlayPage').removeClass('loading');
					});	
				}
				vm.cancelViewSlide();
				vm.cpeCancelViewSlide();
			},
			//eNB Add Policy Task
			addTaskBtnClick() { 
				var vm = this,
					rd = Math.random(),
					url = '${ctx}/plugandplay/policy/goAddConfig.action?r='+rd;

				if(vm.activeTab == 'cpe') {
					url = '${ctx}/plugandplay/policy/goAddConfigCpe.action?r='+rd;
				}else{
					//enb 需调用一次详情数据接口
					var params = { policyId: '' };

		            // 获取策略信息
		            axios.post('${ctx}/SON/SelfConfiguration/getSelfConfigurationInfo.action', stringify(params)).then(function(res){ });
				}

				vm.slideTile = '<%=rb.getString("XinZeng")%> Policy';
				vm.slideURL = url;
				vm.slideHeader = false;
				vm.footerShow = false;
				vm.$refs.slider.showSlide(function(){
					eventBus.$emit('init-plugPlayConfig','add','',vm.activeTab);
				});
				vm.cancelViewSlide();
				vm.cpeCancelViewSlide();
			},
			//eNB Add or modify Policy Save 
			addConfig() {
				eventBus.$emit('save-plugPlayConfig');
			},
			//eNB Add or modify Policy close
			closeConfig() {
				var vm = this;				
				eventBus.$emit('handle-plugPlayCancel')
			},
			//refresh  eNB,CPE table
			reloadConfig() {
				var vm = this,
					code = vm.activeTab == 'enb'? 'enbConfig':'cpeConfig';

				vm.$refs[code].refresh();
			},
			//eNB，CPE rowClick
			rowClick(row) {
				this.currentRow = row;
			},
			//eNB Policy list search
			queryEnb(text) {
				this.enbQuery.searchText = text;
			},
			//eNB Execute Status search
			queryEnbStatus(text) {
				this.enbStatusQuery.searchText = this.statusSearchText;
			},
			//eNB time search
	    	dateChange(val) {
				var vm = this;
				vm.timeRange = val;
				if(val != null){
    				this.enbStatusQuery.queryStartTime = vm.timeRange[0];
	    			this.enbStatusQuery.queryEndTime = vm.timeRange[1];
    			}
			},			
			//eNB policy list loadSuccess
			enbPolicyListSuccess(data) {
				var vm = this,
				properties = data.total || 0;
				vm.enbPolicyNumber = properties || 0;
			},
			//CPE policy list loadSuccess
			cpePolicyListSuccess(data) {
				var vm = this,
				properties = data.total || 0;
				vm.cpePolicyNumber = properties || 0;
			},
			//eNB Execute Status loadSuccess
			enbConfigStatusSuccess(data) {
				var vm = this,
					properties = data.properties||{};

				vm.enbSuccessCount = properties.successCount||0;
				vm.enbFailCount = properties.failCount||0;
			},
			//eNB Execute Status tabs 
			enbTabsChange(val){
				var vm = this;
				vm.enbStatusQuery =  {
					searchText: '',
					productType: '',
					status: '',
					queryStartTime: '',
					queryEndTime: '',
					timeZone: timeZone
				};
				vm.timeRange = [];
				vm.statusSearchText = '';
				
				if(val == '0'){
					vm.enbStatusURL= '${ctx}/SON/SelfConfigurationTask/getSelfConfigTaskPageList.action'; 
				}else {
					vm.enbStatusURL= '${ctx}/SON/SelfConfigurationTask/getSelfConfigTaskRecordPageList.action';
				}
				
				if(val == '1'){
					//升级
					vm.enbStatusQuery.progress = '1';
				}else if(val == '2'){
					//license
					vm.enbStatusQuery.progress = '2';
				}else if(val == '3'){
					//参数配置
					vm.enbStatusQuery.progress = '3';
				}
				
				vm.cancelViewSlide();
				vm.cpeCancelViewSlide();
			},
			//eNB，CPE enable switch click
			switchChange(fn) {
				var vm = this, params = {},
					code = vm.activeTab == 'enb'? 'enbConfig':'cpeConfig';
					url = vm.activeTab == 'enb' ? '${ctx}/SON/SelfConfiguration/updateSelfConfigurationInfo.action' : '${ctx}/plugandplay/policy/updatePolicySwitch.action';

				vm.$nextTick(function(){
					if(vm.activeTab == 'enb'){
						if(vm.currentRow.selfStartEnable == '0'){
							params.selfStartEnable= '1';
						}else{
							params.selfStartEnable= '0';
						}
					}else{
						//CPE
						params.policySwitch = !vm.currentRow.policySwitch;
					}
					
					params.policyId= vm.currentRow.policyId;
				
					axios.post(url, stringify(params)).then(function(res){
						var data = res.data;

						if(data.success) {
							fn();
							vm.$message({
								message: '<%=rb.getString("ChengGong")%>',
								type:'success'
							});

							vm.$refs[code].refresh();
						}else {
							vm.$message({
								message: data['message'],
								type: 'error'
							});
						}

					}).catch(function(){})
				})
			},
			//eNB Execute Status: view info
			viewConfigTask(row){
				var vm = this;
				vm.params_progress.task_id = row.task_id;
				vm.taskUrl = "${ctx}/SON/SelfConfigurationTask/getSelfConfigTaskRecordPageList.action";
				vm.$refs.slideView.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    });
			},
			//eNB Execute Status: retry
			restartStatus(row){
				var vm = this, params = {},
					code = vm.activeTab == 'enb'? 'enbStatus':'cpeStatus';
					url = vm.activeTab == 'enb' ? '${ctx}/SON/SelfConfigurationTask/reExeSelfConfigTask.action' : '${ctx}/plugandplay/task/reExecuteTask.action';

				if(vm.activeTab == 'enb'){
					params = {
						taskId: row.task_id,
						serialNumber: row.serial_number,
						execute_type: row.execute_type
					};
				}else{
					//cpe
					params = { id: row.id };
				}
				vm.retryLoading = row.policyId;				
				axios.post(url, stringify(params)).then(function(res){
					var data = res.data;

					if(data['success']) {
						vm.$message({
							message: data['message'],
							type: 'success'
						});

						vm.$refs[code].refresh();
					}else {
						vm.$message({
							message: data['message'],
							type: 'error'
						});
					}
					vm.retryLoading = '';
				}).catch(function(){});
				vm.cancelViewSlide();
				vm.cpeCancelViewSlide();
			},
			//retry按钮点击
			enbCpeBatchRetryBtn(){
				var vm = this;
				if(vm.enbDeviceSelection.length == 0){
					return
				}else{
					vm.enbCpeBatchRetryForm.isContainSuccessTask = '0';
					vm.enbCpeShoBatchRetryShow = true;
				}
			},
			//确认 retry
			enbCpeBatchRetryConfirm(){
				var vm = this, params = {}, taskIds=[]
					code = vm.activeTab == 'enb'? 'enbStatus':'cpeStatus';
					url = vm.activeTab == 'enb' ? '${ctx}/SON/SelfConfigurationTask/reExeSelfConfigTask.action' : '${ctx}/plugandplay/task/reExecuteTask.action';

				
				if(vm.enbDeviceSelection.length == 0){
					return
				}else{
					if(vm.activeTab == 'enb'){
						taskIds = vm.enbDeviceSelection.map(function(item){ return item.task_id});
						params = {
							taskId: taskIds.join(',')
						};
					}else{
						//cpe
						taskIds = vm.enbDeviceSelection.map(function(item){ return item.id});
						params = {
							 id: taskIds.join(','),
						};
					}
				}
				params.isContainSuccessTask = vm.enbCpeBatchRetryForm.isContainSuccessTask;

				axios.post(url, stringify(params)).then(function(res){
					var data = res.data;

					if(data['success']) {
						vm.$message({
							message: data['message'],
							type: 'success'
						});

						vm.$refs[code].refresh();
					}else {
						vm.$message({
							message: data['message'],
							type: 'error'
						});
					}
				}).catch(function(){});

				vm.cancelViewSlide();
				vm.cpeCancelViewSlide();

				vm.closeEnbCpeBatchRetry();
			},
			closeEnbCpeBatchRetry(){
				var vm = this;
				
				vm.enbCpeBatchRetryForm.isContainSuccessTask = '0';
				vm.enbCpeShoBatchRetryShow = false;

				vm.enbDeviceSelection = [];
				if(vm.activeTab == 'enb'){
					vm.$refs["enbStatus"].clearSelection();	
				}else{
					vm.$refs["cpeStatus"].clearSelection();
				}
			},
			
			//eNB Execute status: start
			startOperStatus(row){
				var vm = this, code = vm.activeTab == 'enb'? 'enbStatus':'cpeStatus',
					params = {
						taskId: row.task_id,
						serialNumber: row.serial_number
					}
				axios.post("${ctx}/SON/SelfConfigurationTask/exeNextSelfConfigTask.action",stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$refs[code].refresh();
					}else{
						vm.$message.error(data["message"]);
					}
				})
				vm.cancelViewSlide();
				vm.cpeCancelViewSlide();
			},
			//eNB Execute Status: delete
			deletStatus(row){
				var vm = this, params = {}, 
					code = vm.activeTab == 'enb'? 'enbStatus':'cpeStatus',
					url = vm.activeTab == 'enb' ? '${ctx}/SON/SelfConfigurationTask/delSelfConfigTask.action' : '${ctx}/plugandplay/task/delAutoPolicyTask.action';

				if(vm.activeTab == 'enb'){
					params = { taskId: row.task_id };
				}else{
					//cpe
					params = { id: row.id };
				}
				vm.$confirm('<%=rb.getString("QueRenShanChu") %>','<%=rb.getString("QueRen") %>').then(function(r){
					if(r) {
						axios.post(url, stringify(params)).then(function(res){
							var data = res.data;

							if(data['success']) {
								vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type: 'success'
								});

								vm.$refs[code].refresh();
							}else {
								vm.$message({
									message: data['message'],
									type: 'error'
								});
							}
							vm.activeTab == 'enb' ? vm.cancelViewSlide() : vm.cpeCancelViewSlide();
							vm.activeTab == 'enb' ? vm.$refs["enbStatus"].clearSelection() : vm.$refs["cpeStatus"].clearSelection();
							//清除已选数据
							vm.enbDeviceSelection = [];
							vm.bulkSelectShow = false;
						}).catch(function(){});
					}
				}).catch(function(){});
			},
			
			//---------------------------------------------------CPE
			//CPE Execute Status: view info
			cpeViewConfigTask(row){
				var vm = this;
				vm.cpe_params_progress.taskId = row.id;
				vm.cpeTaskUrl = "${ctx}/plugandplay/task/queryAutoPolicyTaskRecordPageList.action";
				vm.$refs.cpeSlideView.showSlide(function(){
	    	    	vm.slideModal = false;
	    	    });
			},
			//CPE Policy list search
			queryCpe(text) {
				this.cpeQuery.searchText = text;
			},
			//CPE Execute Status search
			queryCpeStatus(text) {
				this.cpeStatusQuery.searchText = this.cpeStatusSearchText;
			},
			//CPE time search
	    	cpeDateChange(val) {
				var vm = this;
				vm.cpeTimeRange = val;
				if(val != null){
    				this.cpeStatusQuery.queryStartTime = vm.cpeTimeRange[0];
	    			this.cpeStatusQuery.queryEndTime = vm.cpeTimeRange[1];
    			}
			},
			//CPE Execute Status tabs 
			cpeTabsChange(val){
				var vm = this;
				vm.cpeStatusQuery =  {
					searchText: '',
					productType: '',
					status: '',
					queryStartTime: '',
					queryEndTime: '',
					timeZone: timeZone
				};
				if(val == '0'){
					vm.cpeStatusURL= '${ctx}/plugandplay/task/queryAutoPolicyInfoCPEPageList.action';
				}else{
					vm.cpeStatusURL= '${ctx}/plugandplay/task/queryAutoPolicyTaskRecordPageList.action';
				} 
				if(val == '1'){
					//升级
					vm.cpeStatusQuery.taskType = 'upgrade';
				}else if(val == '3'){
					//参数配置
					vm.cpeStatusQuery.taskType = 'config';
				}
				
				vm.cpeTimeRange = [];
				vm.cpeStatusSearchText = '';
				vm.cancelViewSlide();
				vm.cpeCancelViewSlide();
			},
		 	// CPE Execute Status table loadSuccess
		    cpeConfigStatusSuccess(data) {
				var vm = this,
					properties = data.properties || 0;

				vm.cpeSuccessCount = properties.successCount || 0;
				vm.cpeFailCount = properties.failCount || 0;
			},
										
			//-------------------------------------------------------------------eNB,CPE table operation
			optClick(row, evt) {
				var vm = this, modifyFlag, delFlag;
				//开关打开时， 不可修改，不可删除  
				if(row.selfStartEnable == '1' || row.policySwitch == '1'){
					modifyFlag = true;
					delFlag = true;
				}else{
					modifyFlag = false;
					delFlag = false;
				}
				if(vm.activeTab == 'enb') {
					vm.enbMenus= [
						{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info', row: row},
						{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit hidden CODE_PLUG_AND_PLAY",code:'modify', row: row, disable: modifyFlag},
						{label:'<%=rb.getString("JianChe")%>',cls:"el-icon el-icon-operation-scan hidden CODE_PLUG_AND_PLAY",code:'check', row: row},
						{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete hidden CODE_PLUG_AND_PLAY",code:'del', row: row, disable: delFlag}
					];
				}

				if(vm.activeTab=='cpe') {
					vm.cpeMenus= [
						{label:'<%=rb.getString("XinXi")%>',cls:"el-icon el-icon-operation-info",code:'info', row: row},
						{label:'<%=rb.getString("XiuGai")%>',cls:"el-icon el-icon-operation-edit hidden CODE_PLUG_AND_PLAY",code:'modify', row: row, disable: modifyFlag},
						{label:'<%=rb.getString("ShanChu")%>',cls:"el-icon el-icon-operation-delete hidden CODE_PLUG_AND_PLAY",code:'del', row: row, disable: delFlag}
					];
				}

		    	vm.$nextTick(function(){
		    		document.body.click();
					vm.activeTab=='enb' && vm.$refs.enbMenu.show(evt);
					vm.activeTab=='cpe' && vm.$refs.cpeMenu.show(evt);
		    	});
			},
			//eNB policy menu click
			enbMenuClick(row, evt) {
				var vm = this,
					code = row.code,
					data = row.row,
					actions = {
						info: vm.view,
						modify: vm.modify,
						del: vm.deleteData,
						check: vm.checkPolicy
					};

				if(actions[code]) {
					actions[code](data);
				}
			},
			//CPE policy menu operation click
			cpeMenuClick(row, evt) {
				var vm = this,
					code = row.code,
					data = row.row,
					actions = {
						info: vm.view,
						modify: vm.modify,
						del: vm.deleteData
					};

				if(actions[code]) {
					actions[code](data);
				}
			},
			//policy info
			view(row) {
				var vm = this,
					rd = Math.random(),
					url = vm.activeTab == 'enb' ? '${ctx}/plugandplay/policy/goAddConfig.action?r='+rd : '${ctx}/plugandplay/policy/goAddConfigCpe.action?r='+rd,
					params = { policyId: row.policyId };

				//vm.slideTile = '<%=rb.getString("XinXi")%>';
				vm.slideURL = url;
				vm.slideHeader = false;
				vm.footerShow = false;
				vm.$refs.slider.showSlide(params,function(){
					eventBus.$emit('init-plugPlayConfig','readonly', row.policyId,vm.activeTab);
				});
				vm.cancelViewSlide();
				vm.cpeCancelViewSlide();
			},
			//policy modify
			modify(row){
				var vm = this, rd = Math.random(),
					url = vm.activeTab == 'enb' ? '${ctx}/plugandplay/policy/goAddConfig.action?r='+rd : '${ctx}/plugandplay/policy/goAddConfigCpe.action?r='+rd,
					params = { policyId: row.policyId };
	
				//vm.slideTile = '<%=rb.getString("XinXi")%>';
				vm.slideURL = url;
				vm.slideHeader = false;
				vm.footerShow = false;
				vm.$refs.slider.showSlide(params,function(){
					eventBus.$emit('init-plugPlayConfig','modify', row.policyId,vm.activeTab);
				});
				vm.cancelViewSlide();
				vm.cpeCancelViewSlide();
			},
			//policy check
			checkPolicy(row){
				var vm = this,
					params = {policyId: row.policyId};

				vm.ckDLShow = true;
				if(vm.$refs.addListForm){
				    vm.$refs.addListForm.resetFields();
				}

				Object.assign(vm.detectParams, params);
				vm.detectSelections = [];
				vm.detectParams.searchText = '';
				vm.detectSearchText = '';

				vm.$nextTick(function(){
					//清除选中行
					vm.$refs.detectTb.clear();
					vm.detectURL = '${ctx}/SON/SelfConfiguration/queryPolicyCellInfoPageList.action';
				})
			},
			sendDetection() {
				var vm = this,
					params = {
						policyId: vm.currentRow.policyId,
						smallCellCode: vm.addListForm.serialNumber
					};
				
				// 下发检测
				vm.$refs.addListForm.validate((valid) => {
					if(valid){
						axios.post('${ctx}/SON/SelfConfiguration/exeSelfConfigDetectPolicy.action', stringify(params)).then(function(res){
							var data = res.data;
							
							if(data['success']) {
								vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type:'success'
								});
		
								vm.ckDLShow = false;
							}else {
								vm.$message.error(data['message']);
							}
						});
					}
				});
                
			},
			queryDetect(val){
				var vm = this;
				vm.detectParams.searchText = vm.detectSearchText;	
			},

			//ENB batch add 
			enbAddBatchSnClick(){
				var vm = this;

				vm.batchSnDialog = true;
				//清空表单
				if(vm.$refs.batchSnForm){
					vm.$refs.batchSnForm.resetFields();
				}
			},
			
			saveBatchSn(){
				var vm = this,  
					params = {}, 
					snStr = vm.batchSnForm.serialNumber || '',
					list = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});
				
				params.sns = list.join(",");
				params.policyId = vm.currentRow.policyId;

				vm.$refs.batchSnForm.validate((valid) => {
					if(valid){
						axios.post('${ctx}/SON/SelfConfiguration/querySelectedPolicyCellInfos.action', stringify(params)).then((res)=>{
							var data = res.data;
							
							if(data && data.length > 0){		
								vm.$refs.detectTb.appendCheckedRows(data);
								vm.closeBatchSn();
							}else{
								vm.$message('<%=rb.getString("MeiYouKePiPeiSheBei")%>');
							}							
						})
					}
				})
			},
			closeBatchSn(){
				var vm = this;

				vm.batchSnDialog = false;
				vm.$refs.batchSnForm.resetFields(); 
			},

			selectChange(selection) {
				var vm = this;

				vm.detectSelections = selection;
			},
			//policy delete
			deleteData(row) {
				var vm = this,
					url = vm.activeTab == 'enb' ? '${ctx}/SON/SelfConfiguration/delSelfConfigPolicy.action' : '${ctx}/plugandplay/policy/delAutoPolicy.action',
					code = vm.activeTab == 'enb'? 'enbConfig':'cpeConfig',
					params = { policyId: row.policyId };

				vm.$confirm('<%=rb.getString("QueRenShanChu") %>','<%=rb.getString("QueRen") %>').then(function(r){
					if(r) {
						axios.post(url,stringify(params)).then(function(res){
							var data = res.data;

							if(data['success']) {
								vm.$message({
									message: '<%=rb.getString("ChengGong")%>',
									type: 'success'
								});

								vm.$refs[code].refresh();
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
			handerClose() {
				var vm = this;

				if(vm.$refs.enbMenu) vm.$refs.enbMenu.hide();
				if(vm.$refs.cpeMenu) vm.$refs.cpeMenu.hide();
			},			
			// Execute Status rowClick view 
			cancelViewSlide(){ 
		    	var vm = this;
		    	if(vm.$refs.slideView) vm.$refs.slideView.hide();
		    },
		    cpeCancelViewSlide(){ 
		    	var vm = this;
		    	if(vm.$refs.cpeSlideView) vm.$refs.cpeSlideView.hide();
		    },
		    // slide close
		    plugPlayCancelSlide(){
		    	var vm = this;
		    	//vm.reloadConfig();
				if(vm.$refs.slider) vm.$refs.slider.hide();
			},
			//----------------------------------------------------------- 已选 -----------------------------------------------------------
			//<!--status:2-进行中，3-未执行（等待）表格操作项禁止点击 
			enbIsDisabled(row,index){
				if(row.status == '2' || row.status =='3'){
					return false
				}else{
					return true
				}
			},
			//<!--status:2-进行中，3-未执行（等待）表格操作项禁止点击 
			cpeIsDisabled(row,index){
				if(row.status == 'running' || row.status =='waitting'){
					return false
				}else{
					return true
				}
			},
			/**
			* 列表选中
			* @param selection{Array}   选中数据
			*/
			enbCpeBatchSelect(selection){
				var vm = this;

				if(vm.activeTab == 'enb'){
					vm.enbDeviceSelection = selection.map((item)=>{
						return Object.assign(item,{serial_number: item.serial_number});					
					});
				}else{
					//cpe
					vm.enbDeviceSelection = selection.map((item)=>{
						return Object.assign(item,{serialNumber: item.serialNumber});					
					});
				}
			},
	     	// 打开已选弹窗
			 openBulkSelectTable(){
                var vm = this;
                vm.bulkSelectShow = true
            },
            // 关闭已选弹窗
            closeBulkSelectTable(){
                var vm = this;
                vm.bulkSelectShow = false;
            },
            // 设备已选表格 清空事件
            clearBulkSelected(){
                var vm = this;

				if(vm.activeTab == 'enb'){
					vm.$refs["enbStatus"].clearSelection();	
				}else{
					vm.$refs["cpeStatus"].clearSelection();
				}
                
                vm.bulkSelectShow = false;
            },
            // 设备已选表格 单个删除事件
            delBulkSelected(rows){
				//精简以下逻辑
                var vm = this, tabs = '', rowKey = '';
				
				if(vm.activeTab == 'enb'){
					tabs = 'enbStatus',
					rowKey = 'serial_number';		
				}else{
					tabs = 'cpeStatus',
					rowKey = 'id';
				}

				vm.enbDeviceSelection = vm.enbDeviceSelection.filter((items)=>{
					return items[rowKey] != rows[rowKey]
				});
				var selection = this.$refs[tabs].$refs.ctableInner.store.states.selection,
					irow= selection.filter((items)=>{
						return items[rowKey] == rows[rowKey]
					})[0];
				vm.$refs[tabs].toggleRowSelection(irow,false);
				var idx = vm.$refs[tabs].ckList.indexOf(rows[rowKey]);
				vm.$refs[tabs].ckList.splice(idx,1);
                
				if(vm.enbDeviceSelection.length == 0){
                	vm.bulkSelectShow = false;
                }
            },
		},
		mounted() {
			this.init();
			eventBus.$off('close-config').$on('close-config', this.closeConfig);
			eventBus.$off('reload-config-list').$on('reload-config-list', this.reloadConfig);
			
			eventBus.$off('cancel-plugPlaySlide').$on('cancel-plugPlaySlide',this.plugPlayCancelSlide);

			// 初始化拖拽
			var enbHLine = document.querySelector('.horizontal-line-enb'),
				cpeHline = document.querySelector('.horizontal-line-cpe');

			plugAddListener(enbHLine,"mousedown",onmousedownVENB);
			plugAddListener(cpeHline,"mousedown",onmousedownVCPE);
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
	function plugAddListener(element,type,listener,useCapture){
		element.addEventListener?element.addEventListener(type,listener,useCapture):element.attachEvent("on" + type,listener);
	}
	
	/* 鼠标点击事件 */
	function onmousedownVENB(event) {
		var _self = this;
		var lastY = event.clientY, d = document,
		rowItems = document.querySelectorAll('.flex-row-item-enb'),
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
			if(height < 300) height = 300;
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

	function onmousedownVCPE(event) {
		var _self = this;
		var lastY = event.clientY, d = document,
		rowItems = document.querySelectorAll('.flex-row-item-cpe'),
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
			if(height < 300) height = 300;
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