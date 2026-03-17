<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
#upgradeContentPage .device_item{
	position:relative;
}
#upgradeContentPage .device_item .el-ctable-toolbar {
	padding: 0 !important;
}
#upgradeContentPage .el-icon-goback:before, 
#upgradeContentPage .el-icon-status-upgrading:before{
	color:#fff;
	font-size:14px;
}
#upgradeContentPage .list_item .el-tabs__item{
	font-size:14px;
}
#upgradeContentPage .list_item .el-tabs--top{
	border:none;
}
#upgradeContentPage .list_item{
	position:relative;
}
#upgradeContentPage .list_type{
	position:absolute;
	left:20px;
	top:7px;
	z-index: 999;
}
#upgradeContentPage .task_count{
	position:absolute;
	right:10px;
	top:15px;
}
#upgradeContentPage .queryGroup{
	height:28px;
}
#upgradeContentPage .el-badge{
	position:relative;
}
#upgradeContentPage .el-badge__content{
	position:absolute;
	top:6px;
	right:-4px;
	transform:translateY(-50%) translateX(100%);
	background-color:transparent;
	border-radius:10px;
	color:#fff;
	display:inline-block;
	font-size:10px;
	height:12px;
	line-height:11px;
	padding:0 6px;
	text-align:center;
	white-space:nowrap;
	cursor:default;
	border:1px solid transparent;
}
#upgradeContentPage .el-icon-star-badge:before{
	color:#F3916C;
}
#upgradeContentPage .el-date-editor .el-range__close-icon{
	line-height:20px;
}
#upgradeContentPage .commonTabsTop .el-tabs__header {
	border: 1px solid #D5DCEC;
	border-bottom: 0;
	border-radius: 8px 8px 0 0;
	padding: 0 20px;
}
#upgradeContentPage .list_item .el-tabs__header {
	border: 0;
}
#upgradeContentPage .upgradeHeaderBoxCls{
	height: 60px;
	width: 100%;
	background: #FFF;
	display: flex;
	align-items: center;
	justify-content: center;
	margin-bottom: 10px;
}
#upgradeContentPage .upgradeHeaderBoxCls .el-icon::before{
	font-size: 16px;
}
#upgradeContentPage .upgradeHeaderBoxCls .commonRadioButton .el-radio-button__inner{
	display: flex;
	align-items: center
}
#upgradeContentPage .flex-item-cls {
	height: 100%;
	box-sizing: border-box;
}
#upgradeContentPage .upgradeItemBoxCls{
	height: 100%;
	position: relative;
	display: flex;
}
#upgradeContentPage .upgradeItemBoxCls >div{
	overflow: hidden;
}
#upgradeContentPage .upgradeItemBoxCls .upgradeMainPageBox{
	position: relative;
	flex: 1;
}
#upgradeContentPage .upgradeItemBoxCls .importFileBoxCls{
	flex: 0 1 360px;
	margin-left: 10px;
	position: relative;
	background-color: #FFFFFF;
	box-shadow: 0px 0px 10px 1px #E9EDF9;
	border-radius: 10px;
	border: 1px solid #E9EDF9;
	box-sizing: border-box;
}
#upgradeContentPage .newTabs .el-ctable-toolbar{
	padding: 0px!important;
}
#upgradeContentPage .el-radio.is-bordered,
.addDeviceDialog .el-radio.is-bordered{
	height: 30px;
	padding: 7px 20px 0 10px;
}
#upgradeContentPage .el-radio.is-bordered+.el-radio.is-bordered,
.addDeviceDialog .el-radio.is-bordered+.el-radio.is-bordered{
	margin-left: 15px;
}
#upgradeContentPage .el-tabs__header{
	border-bottom: 1px solid #E9E9E9 !important;
}
#upgradeContentPage .rightOutBoxHeadCls{
	height: 50px;
	display: flex;
	align-items: center;
	font-weight: 600;
	font-size: 14px;
	justify-content: space-between;
	padding: 0px 20px;
	border-bottom: 1px solid #E9EDF9;
}
#upgradeContentPage .rightItemMainBox{
	padding: 20px;
}
#upgradeContentPage .rightItemMainBox .el-input,
#upgradeContentPage .rightItemMainBox .el-select{
	width: 100%;
}
#upgradeContentPage .importFileBoxCls .footer{
	width:100%;
	border-top:1px solid #E9E9E9;
	position:absolute;
	bottom:1px;
	height:50px;
	background:#FFFFFF;
	z-index:99;
	display: flex;
	align-items: center;
	border-radius: 0px 0px 10px 10px;
}
#upgradeContentPage .greyIcon::before{
	color: #7A7992;
	font-size: 14px;
}
#upgradeContentPage .labelSlotCls > span{
	color: #999999;
	font-size: 12px;
	margin-left: 10px;
}
#upgradeContentPage .el-radio-button:focus:not(.is-focus):not(.is-disabled){
	-webkit-box-shadow: none!important;
	box-shadow: none!important;
}
#upgradeContentPage .importFileBoxCls .el-select .el-input.is-disabled .el-input__inner,
#upgradeContentPage .importFileBoxCls .el-select .el-input__inner{
    height: unset !important;
}
#upgradeContentPage .importFileBoxCls .moreSelectBoxCls .el-select .el-input.is-disabled .el-input__inner,
#upgradeContentPage .importFileBoxCls .moreSelectBoxCls .el-select .el-input__inner{
    height: 40px !important;
}
#upgradeContentPage .upgradeItemBoxCls .el-ctable-toolbar{
	padding: 0px!important;
}
#upgradeContentPage .importFileBoxCls .el-select .el-tag__close.el-icon-close::before{
    font-size: 9px;
}
</style>
<!-- enb升级 -->
<div class='panelDefault' id="upgradeContentPage" style='border: none;background:unset;overflow: auto;'>
	<!--<div class="upgradeHeaderBoxCls">
		<el-radio-group size="mini" v-model='upgradeHeadType ' class="commonRadioButton" @change="upgradeHeadBtnClick" style='margin: 0 10px 10px;'>
			<el-radio-button v-for="item in upgradeHeadBtnData" :label="item.label"><span :class="item.icon"></span>{{item.label}}</el-radio-button>
		</el-radio-group>
	</div>-->
	<div id="enbUpgrade" class="flex-item-cls" v-show="upgradeHeadType == 'eNB'" style="min-width: 1430px;">
		<div class="upgradeItemBoxCls">
			<div class="upgradeMainPageBox">
				<el-tabs v-model="activeName" style='height:calc(100% - 2px)' class='commonTabsTop'>
					<!-- 升级页面 -->
					<el-tab-pane label='<%=rb.getString("ShengJi")%>&<%=rb.getString("HuiTui")%>' name='upgrade' class='upgrade_item' style='overflow:auto'>
						<div style='flex:1;overflow:hidden;' class='device_item commonTableBorder'>
							<el-ctable v-show="product_type != 'CR-B4860/EU' && product_type != 'CR-B4860/RU' "ref="upgrade_cell_table" id="upgrade_cell_table" row-key="small_cell_code" :url="deviceUrl" :time="6" :height="height" :query-params="query_cell_params" pagination="true" @selection-change="selectCell">
								<el-table-column v-if="upgradeTaskAddBtnShow" type='selection' width="45" reserve-selection=true></el-table-column>
								<el-table-column prop="connection_status" width="50">
									<template slot-scope="scope">
										<div :class="{
											'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
											'':scope.row.have_connected==2,
											'conn_exc':scope.row.connection_status=='Exception',
											'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
									</template>
								</el-table-column>
								<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' width="200"></el-table-column>
								<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>' width="300"></el-table-column>
								<el-table-column prop='rollback_version' label='<%=rb.getString("HuiTuiBanBen")%>' width="200"></el-table-column>
								<el-table-column prop='software_version' label='<%=rb.getString("RuanJianBanBen")%>' width="200"></el-table-column>
								<el-table-column prop='module_type' label='<%=rb.getString("SheBeiXingHaoMing")%>' width="200"></el-table-column>
								<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>'></el-table-column>
								<template slot='toolbar'>
									<div class='toolbarHeadBtnBoxCls commonQuery' style="height:45px;">
										<div class="newIconBoxCls-bt" v-show="upgradeTaskAddBtnShow" style="right:56px;top:12px;" @click="addUpgradeTask" tip="<%=rb.getString("ShengJi")%>">
											<span class='el-icon el-icon-circle-upgrade'></span>
										</div>
										<div class="newIconBoxCls-bt CODE_ENB_ROLLBACK hidden" style="right:20px;top:12px;" @click="addRbTask" tip="<%=rb.getString("HuiTui")%>">
											<span class='el-icon el-icon-circle-restore'></span>
										</div>
										<el-query type="normal"  @query="query" placeholder="<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>"></el-query>
										<div class="tableHeadQueryBoxCls">
											<div v-for="(item,index) in advancedQueryItemList">
												<div v-if="item.type == 'checkbox' && item.isShow" style="margin-right:10px;">
													<el-popfilter
														:label='item.label'
														v-model="item.checkedItemList"
														:list="item.options"
														:visible.sync="item.isShow"
														@check-change="advanceQuery(item.type,item.value,item.checkedItemList)">
													</el-popfilter>
												</div>
												<div v-if="item.type == 'select' && item.isShow" style="margin-right:10px;">
													<el-popfilter
														type="single"
														:label='item.label'
														v-model="item.selectVal"
														:list="item.options"
														:visible.sync="item.isShow"
														@check-change="advanceQuery(item.type,item.value,item.selectVal)">
													</el-popfilter>
												</div>
											</div>
											<div class="advancedQueryItemBox"  style="background: #FFF;" @click="clearFilterClick">
												<%=rb.getString("QingKongShaiXuan")%>
											</div>
										</div>
									</div>
									
									<div style='height:43px;width:100%;background:#FFFFFF;line-height:43px; border-bottom: 1px solid #D5DCEC;'>
										<span style='font-weight:bold;margin-left:10px;margin-right:15px;'><%=rb.getString("ChangPinXingHao")%>:</span>
										<el-radio-group size="mini" v-model='product_type' @change="changeProduct" class='commonRadioButton2'>
											<el-radio-button v-for="item in productTypeList" :label="item.value">{{item.name}}</el-radio-button>
										</el-radio-group>
									</div>
								</template>
							</el-ctable>
							<el-ctable v-show="product_type == 'CR-B4860/EU' " ref="upgrade_eu_table" id="upgrade_eu_table" row-key="serial_number" :url="euDeviceUrl" :time="6" :height="height" :query-params="query_eu_params" pagination="true" @selection-change="selectCell">
								<el-table-column type='selection' width="45" reserve-selection=true></el-table-column>
								<el-table-column label="<%=rb.getString("BUJiZhanBianMa") %>" prop="bu_serial_number"></el-table-column>
								<el-table-column label="<%=rb.getString("LuYouSuoYin") %>" prop="route_index" sortable="true"></el-table-column>
								<el-table-column label="<%=rb.getString("ZhuangTai") %>" prop="status">
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
								<el-table-column label="<%=rb.getString("EUJiZhanBianMa") %>" prop="serial_number"></el-table-column>
								<el-table-column label="<%=rb.getString("EUName") %>" prop="device_name"></el-table-column>
								<el-table-column label="<%=rb.getString("MoKuaiXingHao") %>" prop="model_name"></el-table-column>
								<el-table-column label="<%=rb.getString("RuanJianBanBen") %>" prop="software_version"></el-table-column>
								<template slot='toolbar'>
									<div class='toolbarHeadBtnBoxCls commonQuery' style="height:45px;">
										<div class="newIconBoxCls-bt" v-show="upgradeTaskAddBtnShow" style="right:20px;top:12px;" @click="addUpgradeTask" tip="<%=rb.getString("ShengJi")%>">
											<span class='el-icon el-icon-circle-upgrade'></span>
										</div>

										<el-query type="normal"  @query="queryEU"  placeholder="<%=rb.getString("BUJiZhanBianMa")%>/<%=rb.getString("EUJiZhanBianMa")%>"></el-query>
									</div>
									<div style='height:43px;width:100%;background:#FFFFFF;line-height:43px; border-bottom: 1px solid #D5DCEC;'>
										<span style='font-weight:bold;margin-left:10px;margin-right:15px;'><%=rb.getString("ChangPinXingHao")%>:</span>
										<el-radio-group size="mini" v-model='product_type' @change="changeProduct" class='commonRadioButton2'>
											<el-radio-button v-for="item in productTypeList" :label="item.value">{{item.name}}</el-radio-button>
										</el-radio-group>
									</div>				
								</template>
							</el-ctable>
							<el-ctable v-show="product_type == 'CR-B4860/RU' " ref="upgrade_ru_table" id="upgrade_ru_table" row-key="serial_number" :url="ruDeviceUrl" :time="6" :height="height" :query-params="query_ru_params" pagination="true" @selection-change="selectCell">
								<el-table-column type='selection' width="45" reserve-selection=true></el-table-column>
								<el-table-column label="<%=rb.getString("BUJiZhanBianMa") %>" prop="bu_serial_number"></el-table-column>
								<el-table-column label="<%=rb.getString("LuYouSuoYin") %>" prop="route_index" sortable></el-table-column>
								<el-table-column label="<%=rb.getString("XuLieHao") %>" prop="index"></el-table-column>
								<el-table-column label="<%=rb.getString("ZhuangTai") %>" prop="status">
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
								<el-table-column label="<%=rb.getString("RUJiZhanBianMa") %>" prop="serial_number"></el-table-column>
								<el-table-column label="<%=rb.getString("RUName") %>" prop="device_name"></el-table-column>
								<el-table-column label="<%=rb.getString("MoKuaiXingHao") %>" prop="model_name"></el-table-column>
								<el-table-column label="<%=rb.getString("RuanJianBanBen") %>" prop="software_version"></el-table-column>
								<template slot='toolbar'>
									<div class='toolbarHeadBtnBoxCls commonQuery' style="height:45px;">
										<div class="newIconBoxCls-bt" v-show="upgradeTaskAddBtnShow" style="right:20px;top:12px;" @click="addUpgradeTask" tip="<%=rb.getString("ShengJi")%>">
											<span class='el-icon el-icon-circle-upgrade'></span>
										</div>
										<el-query type="normal"  @query="queryRU" placeholder="<%=rb.getString("BUJiZhanBianMa")%>/<%=rb.getString("RUJiZhanBianMa")%>"></el-query>
									</div>
									<div style='height:43px;width:100%;background:#FFFFFF;line-height:43px;border-bottom: 1px solid #D5DCEC;'>
										<span style='font-weight:bold;margin-left:10px;margin-right:15px;'><%=rb.getString("ChangPinXingHao")%>:</span>
										<el-radio-group size="mini" v-model='product_type' @change="changeProduct" class='commonRadioButton2'>
											<el-radio-button v-for="item in productTypeList" :label="item.value">{{item.name}}</el-radio-button>
										</el-radio-group>
									</div>
								</template>
							</el-ctable>
						</div>
						<div style='flex:1;margin-top:10px;overflow:auto; border: 1px solid #D5DCEC; border-radius: 8px;' class='list_item'>
							<el-tabs v-model="list_name" style='height:100%' class='newTabs'>
								<el-tab-pane label="<%=rb.getString("RuanJianShengJi")%>" name="software" style="position:relative">
									<el-radio-group v-model="list_type" class="list_type commonRadioButton" @change="changeListType" size="mini">
										<el-radio-button label="task"><%=rb.getString("RenWuLieBiao")%></el-radio-button>
										<el-radio-button label="device"><%=rb.getString("BackupRestoreSheBeiLieBiao")%></el-radio-button>
									</el-radio-group>
									<el-ctable v-show="showTask" id="upgrade_task_table" ref="upgrade_task_table" time=6 :url="taskUrl" :height="height" :query-params="query_task_params" pagination="true" @load-success="loadSuccessTask">
										<el-table-column label='' width="30" class-name="no-text-tips">
											<template slot-scope="scope">
												<div class="el-icon el-icon-operation-more" @click="optClickTask(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
											</template>
										</el-table-column>
										<el-table-column prop='TASK_NAME' label='<%=rb.getString("RenWuMingCheng")%>' width="400"></el-table-column>
										<el-table-column prop='CREATE_USER' label='<%=rb.getString("CaoZuoRen")%>' width="100"></el-table-column>
										<el-table-column prop='CREATE_TIME' label='<%=rb.getString("CaoZuoShiJian")%>' width="200"></el-table-column>
										<el-table-column prop='VERSION' label='<%=rb.getString("BanBen")%>' width="200"></el-table-column>
										<el-table-column prop='TYPE' label='<%=rb.getString("ShengJiLeiXing")%>' width="200" :formatter="typeFmt"></el-table-column>
										<el-table-column prop='PRODUCT' label='<%=rb.getString("ChangPinXingHao")%>' width="200"></el-table-column>
										<el-table-column prop="TASK_STATUS" label='<%=rb.getString("ZhuangTai")%>' width="120" >
											<template slot-scope="scope">
												<div v-html="changePasswordTaskTableStatus(scope.row.TASK_STATUS)"></div>
											</template>
										</el-table-column>
										<el-table-column prop="TASK_PROGRESS" label='<%=rb.getString("JinDu")%>' width="100"></el-table-column>
										<el-table-column prop="TASK_RESULT" label='<%=rb.getString("JieGuo")%>' width="100" :formatter="resultFmt"></el-table-column>
										<el-table-column label='<%=rb.getString("BaoLiuPeiZhi")%>' width="200" prop="IS_KEEP_CONFIG" :formatter="configFmt"></el-table-column>
										<el-table-column prop="START_TIME" label='<%=rb.getString("KaiShiShiJian")%>' width="180"></el-table-column>
										<el-table-column prop="END_TIME" label='<%=rb.getString("JieShuShiJian")%>' width="180"></el-table-column>
										<template slot="toolbar">
											<div class='toolbarHeadBtnBoxCls commonQuery' style="height:45px;">
												<el-query type="normal" style='margin-left:198px;' @query="queryTask" placeholder="<%=rb.getString("RenWuMingCheng")%>"></el-query>
												<el-date-picker style='margin-left: 20px;' 
													v-model="dateValue"
													type="datetimerange"
													value-format="yyyy-MM-dd HH:mm:ss"
													range-separator="——"  
													@change="dateChange"
													start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
													end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
												</el-date-picker>
											</div>
											<div class='task_count'>
												<p><span><i class='el-icon el-icon-status-waiting1'></i><%=rb.getString("DengDai")%></span><span>{{waitNum}}</span></p>
												<p><span><i class='el-icon el-icon-status-inProgress'></i><%=rb.getString("JinXingZhong")%></span><span>{{progressNum}}</span></p>
												<p><span><i class='el-icon el-icon-status-suspend'></i><%=rb.getString("ZanTing")%></span><span>{{suspendNum}}</span></p>
												<p><span><i class='el-icon el-icon-status-terminate'></i><%=rb.getString("YiJieShu")%></span><span>{{endNum}}</span></p>
											</div>
										</template>
									</el-ctable>
									<el-ctable v-show="!showTask" ref="upgrade_result_table" id="upgrade_result_table" time=6  :url="resultUrl" :height="height" :query-params="query_result_params" pagination="true" @load-success="loadsuccessResult">
										<el-table-column label='' width="30" class-name="no-text-tips">
											<template slot-scope="scope">
												<div v-if="scope.row.result == '3'" class="el-icon el-icon-operation-restart" @click="restartTask('single',scope.row.taskId,scope.row.smallCellCode)" style="cursor: pointer;"></div>
											</template>
										</el-table-column>
										<el-table-column prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>' width="200"></el-table-column>
										<el-table-column prop='cellName' label='<%=rb.getString("HostName")%>' width="300"></el-table-column>
										<el-table-column prop='taskName' label='<%=rb.getString("RenWuMingCheng")%>' width="300"></el-table-column>
										<el-table-column prop='originalVersion' label='<%=rb.getString("ChuShiBanBen")%>' width="200"></el-table-column>
										<el-table-column prop='upgradeVersion' label='<%=rb.getString("ShengJiBanBen")%>' width="200"></el-table-column>
										<el-table-column prop='upgradeType' label='<%=rb.getString("ShengJiLeiXing")%>' width="200" :formatter="typeFmt"></el-table-column>
										<el-table-column prop='productType' label='<%=rb.getString("ChangPinXingHao")%>' width="200"></el-table-column>
										<el-table-column prop="status" label='<%=rb.getString("ZhuangTai")%>' width="120" >
											<template slot-scope="scope">
												<div v-html="resultTableStatus(scope.row.status)"></div>
											</template>
										</el-table-column>
										<el-table-column prop="result" label='<%=rb.getString("JieGuo")%>' width="100" :formatter="deviceResultFmt"></el-table-column>
										<el-table-column prop='failureReason' label='<%=rb.getString("ShiBaiYuanYin")%>' width="200"></el-table-column>
										<el-table-column prop='startTime' label='<%=rb.getString("KaiShiShiJian")%>' width="200"></el-table-column>
										<el-table-column prop='endTime' label='<%=rb.getString("JieShuShiJian")%>' width="200"></el-table-column>
										<template slot="toolbar">
											<div class='toolbarHeadBtnBoxCls commonQuery' style='padding-left: 210px; height:45px;'>
												<div class="newIconBoxCls-bt" style="right:20px;top:10px;" @click="exportResult" tip="<%=rb.getString("DaoChu")%>">
													<span class='el-icon el-icon-operation-export'></span>
												</div>
												<el-query type="normal" @query="queryResult" placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%> / <%=rb.getString("RenWuMingCheng")%>"></el-query>
												<div style='position: absolute; right: 60px;top:15px;'>
													<p class='suc_count'><span><i class='el-icon el-icon-circle-success'></i><%=rb.getString("ChengGong")%></span><span>{{sucNum}}</span></p>
													<p class='fail_count'><span><i class='el-icon el-icon-circle-close'></i><%=rb.getString("ShiBai")%></span><span>{{failNum}}</span></p>
												</div>
											</div>
										</template>
									</el-ctable>
								</el-tab-pane>
								<el-tab-pane label="<%=rb.getString("ShengJiHuiTuiRenWu")%>" name="rollback">
									<el-radio-group v-model="list_type_rb" class="list_type commonRadioButton" @change="changeListRb" size="mini">
										<el-radio-button label="task"><%=rb.getString("RenWuLieBiao")%></el-radio-button>
										<el-radio-button label="device"><%=rb.getString("BackupRestoreSheBeiLieBiao")%></el-radio-button>
									</el-radio-group>
									<el-ctable v-show="showTaskRb" ref="rb_task_table" id="rb_task_table" time=6 :url="taskUrlRb" :height="height" :query-params="query_task_params_rb" pagination="true" @load-success="loadSuccessTask">
										<el-table-column label='' width="30" class-name="no-text-tips">
											<template slot-scope="scope">
												<div class="el-icon el-icon-operation-more" @click="optClickTask(scope.row,event)" v-clickoutside="handerClose" style="cursor: pointer;"></div>
											</template>
										</el-table-column>
										<el-table-column prop='TASK_NAME' label='<%=rb.getString("RenWuMingCheng")%>' width="400"></el-table-column>
										<el-table-column prop='CREATE_USER' label='<%=rb.getString("CaoZuoRen")%>' width="300"></el-table-column>
										<el-table-column prop='CREATE_TIME' label='<%=rb.getString("CaoZuoShiJian")%>' width="200"></el-table-column>
										<el-table-column prop='PRODUCT' label='<%=rb.getString("ChangPinXingHao")%>' width="200"></el-table-column>
										<el-table-column prop="TASK_STATUS" label='<%=rb.getString("ZhuangTai")%>' width="120" >
											<template slot-scope="scope">
												<div v-html="changePasswordTaskTableStatus(scope.row.TASK_STATUS)"></div>
											</template>
										</el-table-column>
										<el-table-column prop="TASK_PROGRESS" label='<%=rb.getString("JinDu")%>' width="100"></el-table-column>
										<el-table-column prop="TASK_RESULT" label='<%=rb.getString("JieGuo")%>' width="100" :formatter="resultFmt"></el-table-column>
										<el-table-column prop="START_TIME" label='<%=rb.getString("KaiShiShiJian")%>' width="180"></el-table-column>
										<el-table-column prop="END_TIME" label='<%=rb.getString("JieShuShiJian")%>' width="180"></el-table-column>
										<template slot="toolbar">
											<div class='toolbarHeadBtnBoxCls commonQuery' style="height:45px;">
												<el-query type="normal" style='margin-left:198px;' @query="queryTask" placeholder="<%=rb.getString("RenWuMingCheng")%>"></el-query>
												<el-date-picker style='margin-left: 20px;' 
													v-model="dateValueRb"
													type="datetimerange"
													value-format="yyyy-MM-dd HH:mm:ss"
													range-separator="——"  
													@change="dateChangeSoft"
													start-placeholder='<%=rb.getString("KaiShiShiJian")%>' 
													end-placeholder='<%=rb.getString("JieShuShiJian")%>'>
												</el-date-picker>
											</div>
											<div class='task_count'>
												<p><span><i class='el-icon el-icon-status-waiting1'></i><%=rb.getString("DengDai")%></span><span>{{waitNum}}</span></p>
												<p><span><i class='el-icon el-icon-status-inProgress'></i><%=rb.getString("JinXingZhong")%></span><span>{{progressNum}}</span></p>
												<p><span><i class='el-icon el-icon-status-suspend'></i><%=rb.getString("ZanTing")%></span><span>{{suspendNum}}</span></p>
												<p><span><i class='el-icon el-icon-status-terminate'></i><%=rb.getString("YiJieShu")%></span><span>{{endNum}}</span></p>
											</div>
										</template>
									</el-ctable>
									<el-ctable v-show="!showTaskRb" ref="rb_result_table" id="rb_result_table" time=6 :url="resultUrlRb" :height="height" :query-params="query_result_params_rb" pagination="true" @load-success="loadsuccessResult">
										<el-table-column prop='serialNumber' label='<%=rb.getString("XiaoZhanBianMa")%>' width="200"></el-table-column>
										<el-table-column prop='cellName' label='<%=rb.getString("HostName")%>' width="300"></el-table-column>
										<el-table-column prop='taskName' label='<%=rb.getString("RenWuMingCheng")%>' width="300"></el-table-column>
										<el-table-column prop='originalVersion' label='<%=rb.getString("ChuShiBanBen")%>' width="200"></el-table-column>
										<el-table-column prop='productType' label='<%=rb.getString("ChangPinXingHao")%>' width="200"></el-table-column>
										<el-table-column prop="status" label='<%=rb.getString("ZhuangTai")%>' width="120" >
											<template slot-scope="scope">
												<div v-html="resultTableStatus(scope.row.status)"></div>
											</template>
										</el-table-column>
										<el-table-column prop="result" label='<%=rb.getString("JieGuo")%>' width="100" :formatter="deviceResultFmt"></el-table-column>
										<el-table-column prop='failureReason' label='<%=rb.getString("ShiBaiYuanYin")%>' width="200"></el-table-column>
										<el-table-column prop='startTime' label='<%=rb.getString("KaiShiShiJian")%>' width="200"></el-table-column>
										<el-table-column prop='endTime' label='<%=rb.getString("JieShuShiJian")%>' width="200"></el-table-column>
										<template slot="toolbar">
											<div class='toolbarHeadBtnBoxCls commonQuery' style='padding-left: 210px; height:45px;'>
												<div class="newIconBoxCls-bt" style="right:20px;top:10px;" @click="exportResult" tip="<%=rb.getString("DaoChu")%>">
													<span class='el-icon el-icon-operation-export'></span>
												</div>
												<el-query type="normal" @query="queryResult" placeholder="<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%> / <%=rb.getString("RenWuMingCheng")%>"></el-query>
												<div style='position: absolute; right: 60;top:15px;'>
													<p class='suc_count'><span><i class='el-icon el-icon-circle-success'></i><%=rb.getString("ChengGong")%></span><span>{{sucNum}}</span></p>
													<p class='fail_count'><span><i class='el-icon el-icon-circle-close'></i><%=rb.getString("ShiBai")%></span><span>{{failNum}}</span></p>
												</div>
											</div>
										</template>
									</el-ctable>
								</el-tab-pane>
								<el-cmenu ref="menu_task" :data="menus_task" @click="clickMenuTask"></el-cmenu>
								<el-cmenu ref="menu_device" :data="menus_device" @click="clickMenuDevice"></el-cmenu>
							</el-tabs>
						</div>
					</el-tab-pane>
					<!-- 文件页面 -->
					<el-tab-pane label='<%=rb.getString("WenJian")%>' name='file' style='position:relative; '>
						<el-ctable ref="file_table"  :url="file_url" :height="height" :query-params="query_file_params" pagination="true" style='border: 1px solid #D5DCEC;border-top:none;height:calc(100% - 2px); border-radius: 0 0 8px 8px;'>
							<el-table-column label='' width="30" class-name="operationColumn">
								<template slot-scope="scope">
									<div class="el-icon el-icon-operation-more" @click="optClickFile(scope.row,event)" v-clickoutside="handerClose" ></div>
								</template>
							</el-table-column>
							<el-table-column label='<%=rb.getString("BanBen")%>' width="400"  prop="version" show-overflow-tooltip="true">
								<template slot-scope="scope">
									<div v-if="scope.row.recommend == '1'" class="el-badge">
										<span>{{scope.row.version}}</span>
										<span class='el-badge__content el-icon el-icon-star-badge'></span>
									</div>
									<div v-else>{{scope.row.version}}</div>
								</template>
							</el-table-column>
							<el-table-column label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>' width="250" prop="product"  v-if="file_type !== 'ap'"></el-table-column>
							<el-table-column label='<%=rb.getString("WenJianDaXiao")%>' width="200" prop="size"></el-table-column>
							<el-table-column v-if="isCloudCore=='true'?true:false" label='<%=rb.getString("BanBenLeiXing")%>' width="200" prop="toWho" :formatter="towhoFmt"></el-table-column>
							<el-table-column label='<%=rb.getString("ShangChuanShiJian")%>' prop="upload_time"></el-table-column>
							<template slot="toolbar">
								<el-radio-group size="mini" v-model='file_type' class="commonRadioButton" @change="changeFileType" style='margin: 10px 10px;display:inline-block;'>
									<!--<div @click="fileTypeClick">-->
										<el-radio-button label="upgrade"><%=rb.getString("IMAGE")%></el-radio-button>
										<el-radio-button label="ca"><%=rb.getString("CABanBen")%></el-radio-button>
										<el-radio-button  label="fpga"><%=rb.getString("FPGAShengJiWenJian")%></el-radio-button>
										<el-radio-button  label="ap"><%=rb.getString("APShengJiWenJian")%></el-radio-button>
									<!--</div>-->
								</el-radio-group>
								<div class='toolbarHeadBtnBoxCls' style="height:45px;">
									<div class="newIconBoxCls-bt CODE_ENB_UPGRADE_FILE hidden" style="right:20px;top:5px;" @click="importFileClick" tip="<%=rb.getString("DaoRuWenJian")%>">
										<span class='el-icon el-icon-operation-import'></span>
									</div>
									<div class='queryGroup commonSearchWarp'>
										<el-input v-model='query_file_form.searchText' @keyup.enter.native="queryFile" class='pairgrid-query' placeholder='<%=rb.getString("BanBen")%>'></el-input>
										<i @click='queryFile' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
									</div>
								</div>
							</template>
						</el-ctable>
					</el-tab-pane>
					<el-cmenu ref="menu_file" :data="menus_file" @click="clickMenuFile"></el-cmenu>
				</el-tabs>
			</div>
			<div class="importFileBoxCls" v-show="importFileShow">
				<div class="rightOutBoxHeadCls">
					<span>{{rightOutBoxTitle}}</span>
					<span class="el-icon el-icon-close greyIcon" @click="rightBoxClose"></span>
				</div>
				<div class="rightItemMainBox" v-show="file_type == 'upgrade'">
					<el-form ref="importUpgradeFileForm" :model="importUpgradeFileForm" label-position="top" :rules="importUpgradeFileFormRules" :hide-required-asterisk=true>
						<el-form-item prop="product" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" v-if="file_type != 'ap'" key="product" class='moreSelectBoxCls'>
							<el-select v-model='importUpgradeFileForm.productList' multiple collapse-tags :disabled="importViewFlag" @change="productListChange">	
								<el-option v-for='item in productTypeList' :key="item.value" :label="item.name" :value="item.value"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="<%=rb.getString("WenJianMing")%>" prop="fileName">
							<span slot="label" class="labelSlotCls">
								<%=rb.getString("WenJianMing")%>
                                <span v-show="importFileType == 'add' && upgradeFileType == 'tar.gz'" >( <%=rb.getString("ShengJiZhiChiGeShi")%> )</span>
                                <span v-show="importFileType == 'add' && upgradeFileType == 'IMG,EXT'" >( <%=rb.getString("ZhiZhiChiIMGHeEXTGeShi")%> )</span>
                                <span v-show="importFileType == 'add' && upgradeFileType == 'IMG'" >( <%=rb.getString("ZhiChiIMGGeShi")%> )</span>
							</span>
							<el-upload 
								v-show="importFileType == 'add'" 
								:before-upload='beforeUpload'  
								:on-success='checkImportFile' 
								:on-change="importFileChange" 
								:show-file-list=false 
								ref="importUpgradeFile" 
								:action="importUpgradeFileForm.importFileUrl"
								:auto-upload="false">
								<el-input :readonly="true" :value="importUpgradeFileForm.fileName">
									<a slot="append" class="el-icon el-icon-operation-import greyIcon" @click="importFileSelect"></a>
								</el-input>
								<a slot="trigger" ref="file_up_upgrade"></a>
							</el-upload>
							<el-input v-show="importFileType != 'add'" v-model='importUpgradeFileForm.fileName' :disabled="true"></el-input>
						</el-form-item>
						<el-form-item label="<%=rb.getString("BanBen")%>" prop="version">
							<el-input v-model='importUpgradeFileForm.version' :disabled="importViewFlag"></el-input>
						</el-form-item>
						<el-form-item label='<%=rb.getString("Title_KaiFangYunYingShang")%>' prop='to_who' v-if="showToWho">
							<el-select v-model='importUpgradeFileForm.to_who' :disabled="importViewFlag">
								<el-option label='<%=rb.getString("ShangYongBanBen")%>' value='all'></el-option>
								<el-option label='<%=rb.getString("CeShiBanBen")%>' value='none'></el-option>
								<el-option label='<%=rb.getString("BetaBanBen")%>' value="beta"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label='<%=rb.getString("TuiJian")%>' prop='recommend'>
							<el-select v-model='importUpgradeFileForm.recommend' :disabled="importViewFlag">
								<el-option label='<%=rb.getString("Shi")%>' value='1'></el-option>
								<el-option label='<%=rb.getString("Fou")%>' value='0'></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="<%=rb.getString("MiaoShu")%>" prop='desc'>
							<el-input v-model='importUpgradeFileForm.desc' type='textarea' :rows='2' :disabled="importViewFlag"></el-input>
						</el-form-item>
					</el-form>
				</div>
                <div class="rightItemMainBox" v-show="file_type == 'ca'">
					<el-form ref="importCaFileForm" :model="importCaFileForm" label-position="top" :rules="importCaFileFormRules" :hide-required-asterisk=true>
						<el-form-item prop="product" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" key="product">
							<el-select v-model='importCaFileForm.product' :disabled="importViewFlag" @change="productListChange">	
								<el-option v-for='item in productTypeList' :label="item.name" :value="item.value"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="<%=rb.getString("WenJianMing")%>" prop="fileName">
							<span slot="label" class="labelSlotCls">
								<%=rb.getString("WenJianMing")%>
                                <span v-show="importFileType == 'add'" >( <%=rb.getString("ZhiChiPatchGeShi")%> )</span>
							</span>
							<el-upload 
								v-show="importFileType == 'add'" 
								:before-upload='beforeUpload'  
								:on-success='checkImportFile' 
								:on-change="importFileChange" 
								:show-file-list=false 
								ref="importCaFile" 
								:action="importCaFileForm.importFileUrl"
								:auto-upload="false">
								<el-input :readonly="true" :value="importCaFileForm.fileName">
									<a slot="append" class="el-icon el-icon-operation-import greyIcon" @click="importFileSelect"></a>
								</el-input>
								<a slot="trigger" ref="file_up_ca"></a>
							</el-upload>
							<el-input v-show="importFileType != 'add'" v-model='importCaFileForm.fileName' :disabled="true"></el-input>
						</el-form-item>
						<el-form-item label="<%=rb.getString("BanBen")%>" prop="version">
							<el-input v-model='importCaFileForm.version' :disabled="importViewFlag"></el-input>
						</el-form-item>
						<el-form-item label='<%=rb.getString("Title_KaiFangYunYingShang")%>' prop='to_who' v-if="showToWho">
							<el-select v-model='importCaFileForm.to_who' :disabled="importViewFlag">
								<el-option label='<%=rb.getString("ShangYongBanBen")%>' value='all'></el-option>
								<el-option label='<%=rb.getString("CeShiBanBen")%>' value='none'></el-option>
								<el-option label='<%=rb.getString("BetaBanBen")%>' value="beta"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label='<%=rb.getString("TuiJian")%>' prop='recommend'>
							<el-select v-model='importCaFileForm.recommend' :disabled="importViewFlag">
								<el-option label='<%=rb.getString("Shi")%>' value='1'></el-option>
								<el-option label='<%=rb.getString("Fou")%>' value='0'></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="<%=rb.getString("MiaoShu")%>" prop='desc'>
							<el-input v-model='importCaFileForm.desc' type='textarea' :rows='2' :disabled="importViewFlag"></el-input>
						</el-form-item>
					</el-form>
				</div>
                <div class="rightItemMainBox" v-show="file_type == 'fpga'">
					<el-form ref="importFpgaFileForm" :model="importFpgaFileForm" label-position="top" :rules="importFpgaFileFormRules" :hide-required-asterisk=true>
						<el-form-item prop="product" label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" key="product">
							<el-select v-model='importFpgaFileForm.product' :disabled="importViewFlag" @change="productListChange">	
								<el-option v-for='item in productTypeList' :label="item.name" :value="item.value"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="<%=rb.getString("WenJianMing")%>" prop="fileName">
							<span slot="label" class="labelSlotCls">
								<%=rb.getString("WenJianMing")%>
                                <span v-show="importFileType == 'add'" >( <%=rb.getString("ZhiChiIMGGeShi")%> )</span>
							</span>
							<el-upload 
								v-show="importFileType == 'add'" 
								:before-upload='beforeUpload'  
								:on-success='checkImportFile' 
								:on-change="importFileChange" 
								:show-file-list=false 
								ref="importFpgaFile" 
								:action="importFpgaFileForm.importFileUrl"
								:auto-upload="false">
								<el-input :readonly="true" :value="importFpgaFileForm.fileName">
									<a slot="append" class="el-icon el-icon-operation-import greyIcon" @click="importFileSelect"></a>
								</el-input>
								<a slot="trigger" ref="file_up_fpga"></a>
							</el-upload>
							<el-input v-show="importFileType != 'add'" v-model='importFpgaFileForm.fileName' :disabled="true"></el-input>
						</el-form-item>
						<el-form-item label="<%=rb.getString("BanBen")%>" prop="version">
							<el-input v-model='importFpgaFileForm.version' :disabled="importViewFlag"></el-input>
						</el-form-item>
						<el-form-item label='<%=rb.getString("Title_KaiFangYunYingShang")%>' prop='to_who' v-if="showToWho">
							<el-select v-model='importFpgaFileForm.to_who' :disabled="importViewFlag">
								<el-option label='<%=rb.getString("ShangYongBanBen")%>' value='all'></el-option>
								<el-option label='<%=rb.getString("CeShiBanBen")%>' value='none'></el-option>
								<el-option label='<%=rb.getString("BetaBanBen")%>' value="beta"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label='<%=rb.getString("TuiJian")%>' prop='recommend'>
							<el-select v-model='importFpgaFileForm.recommend' :disabled="importViewFlag">
								<el-option label='<%=rb.getString("Shi")%>' value='1'></el-option>
								<el-option label='<%=rb.getString("Fou")%>' value='0'></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="<%=rb.getString("MiaoShu")%>" prop='desc'>
							<el-input v-model='importFpgaFileForm.desc' type='textarea' :rows='2' :disabled="importViewFlag"></el-input>
						</el-form-item>
					</el-form>
				</div>
                <div class="rightItemMainBox" v-show="file_type == 'ap'">
					<el-form ref="importApFileForm" :model="importApFileForm" label-position="top" :rules="importApFileFormRules" :hide-required-asterisk=true>
						<el-form-item label="<%=rb.getString("WenJianMing")%>" prop="fileName">
							<span slot="label" class="labelSlotCls">
								<%=rb.getString("WenJianMing")%>
                                <span v-show="importFileType == 'add'" >( <%=rb.getString("ZhiChiIMGGeShi")%> )</span>
							</span>
							<el-upload 
								v-show="importFileType == 'add'" 
								:before-upload='beforeUpload'  
								:on-success='checkImportFile' 
								:on-change="importFileChange" 
								:show-file-list=false 
								ref="importApFile" 
								:action="importApFileForm.importFileUrl"
								:auto-upload="false">
								<el-input :readonly="true" :value="importApFileForm.fileName">
									<a slot="append" class="el-icon el-icon-operation-import greyIcon" @click="importFileSelect"></a>
								</el-input>
								<a slot="trigger" ref="file_up_ap"></a>
							</el-upload>
							<el-input v-show="importFileType != 'add'" v-model='importApFileForm.fileName' :disabled="true"></el-input>
						</el-form-item>
						<el-form-item label="<%=rb.getString("BanBen")%>" prop="version">
							<el-input v-model='importApFileForm.version' :disabled="importViewFlag"></el-input>
						</el-form-item>
						<el-form-item label='<%=rb.getString("Title_KaiFangYunYingShang")%>' prop='to_who' v-if="showToWho">
							<el-select v-model='importApFileForm.to_who' :disabled="importViewFlag">
								<el-option label='<%=rb.getString("ShangYongBanBen")%>' value='all'></el-option>
								<el-option label='<%=rb.getString("CeShiBanBen")%>' value='none'></el-option>
								<el-option label='<%=rb.getString("BetaBanBen")%>' value="beta"></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label='<%=rb.getString("TuiJian")%>' prop='recommend'>
							<el-select v-model='importApFileForm.recommend' :disabled="importViewFlag">
								<el-option label='<%=rb.getString("Shi")%>' value='1'></el-option>
								<el-option label='<%=rb.getString("Fou")%>' value='0'></el-option>
							</el-select>
						</el-form-item>
						<el-form-item label="<%=rb.getString("MiaoShu")%>" prop='desc'>
							<el-input v-model='importApFileForm.desc' type='textarea' :rows='2' :disabled="importViewFlag"></el-input>
						</el-form-item>
					</el-form>
				</div>
				<div class="footer" v-show="importFileType !== 'view'" >
					<div class="lnkbuttonGroup"  style="margin-left:20px;" >
						<el-button type="primary" @click="importFileSubmit"><%=rb.getString("QueDing")%></el-button>
						<el-button @click="rightBoxClose"><%=rb.getString("QuXiao")%></el-button>
					</div>
				</div>
			</div>
		</div>
		<el-slide ref="slide" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :header='slideHeader' :position="slidePosition" class='commonBorderSlide'
		:height="slideHeight" :modal='slideModal'  :width="slideWidth" :subloading="slideSubmitLoading" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'"  @cancel='cancelSlide' @ok='saveSlide'>
		</el-slide>
		<el-tslide class='tslide' ref="tslide" :url="slideUrl" :title="slideTitle" :footer="false" :position="slidePosition"
						:height="slideHeight" :modal='modal'  :width="slideWidth" @ok='' @cancel='cancalImportFile' @operate='' :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
		</el-tslide>
	</div>
	<div id="gnbUpgrade" class="flex-item-cls" v-show="upgradeHeadType == 'gNB'"></div>
	<div id="cpeUpgrade" class="flex-item-cls" v-show="upgradeHeadType == 'CPE'"></div>
	<div id="egwUpgrade" class="flex-item-cls" v-show="upgradeHeadType == 'eGW'"></div>

	
</div>
<script>
var enbFileVue = new Vue({
	el:'#upgradeContentPage',
	data(){
		var vm = this,
			versionValidate = function(rule,value,callback) {
				if(value) {
					if(value.length>45) {
						callback('<%=rb.getString("FanWei")%>:1-45 <%=rb.getString("ZiFuFuShu")%>');
					}else {
						if(vm.importFileType == 'add'){
							axios.get("${ctx}/cell/version/verifyUVExist.action",{
								params:{
									fileType : vm.file_type,
									version : value
								}
							}).then(function(response){
								var data = response.data;
								if(data){
									callback('<%=rb.getString("Msg_ShengJiWenJianBanBenYiJingCunZai")%>');
								}else{
									callback();
								}
							})
						}else{
							callback();
						}
					}
				}else {
					callback('<%=rb.getString("QingShuRuWenJianBanBen")%>');
				}
			},
			fileNameValidate = (rule,value,callback) => {
                if(vm.file_type == 'upgrade'){
                    if(vm.upgradeFileType == 'tar.gz'){
                        var reg = vm.difFileObj[vm.file_type].fileFmtYD;
                        var message = vm.difFileObj[vm.file_type].fileErrorMsgYD;
                    }else if(vm.upgradeFileType == 'IMG,EXT'){
                        var reg = vm.difFileObj[vm.file_type].fileFmt4860;
                        var message = vm.difFileObj[vm.file_type].fileErrorMsg4860;
                    }else if(vm.upgradeFileType == 'IMG'){
                        var reg = vm.difFileObj[vm.file_type].fileFmt;
                        var message = vm.difFileObj[vm.file_type].fileErrorMsg;
                    }
                    if(vm.upgradeFileType){
                        if(value == "" || value == null  || value == undefined){
                            callback(new Error("<%=rb.getString("QingXianXuanZeWenJian")%>"))
                        }else if(!vm.fileFormatMatch(value,reg)){
                            callback(new Error(message))
                        }else{
                            var pathSplit = value.split(/\\/),
                                filename = pathSplit[pathSplit.length - 1];
                            
                            if(filename.length>100) {
                                callback("<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>");
                            }else {
                                callback();
                            }
                        }
                    }else{
                        callback();
                    }
                }else{
					var reg = vm.difFileObj[vm.file_type].fileFmt;
					var message = vm.difFileObj[vm.file_type].fileErrorMsg;

                    if(value == "" || value == null  || value == undefined){
                        callback(new Error("<%=rb.getString("QingXianXuanZeWenJian")%>"))
                    }else if(!vm.fileFormatMatch(value,reg)){
                        callback(new Error(message))
                    }else{
                        var pathSplit = value.split(/\\/),
                            filename = pathSplit[pathSplit.length - 1];
                        
                        if(filename.length>100) {
                            callback("<%=rb.getString("WenJianMingBuNengChaoGuoYaoQiu")%>");
                        }else {
                            callback();
                        }
                    }
				}
			},
            productValidate = (rule,value,callback) => {
                
				if(value == "" || value == null  || value == undefined){
					callback(new Error("<%=rb.getString("ShuRuBiTianXiang")%>"))
				}else{
                    if(vm.upgradeFileType){
                        callback()
                    }else{
                        callback(new Error("<%=rb.getString("MeiYouDuiYingDeShengJiWenJian")%>"))
                    }
				}
			};
		return{
			activeName : 'upgrade',
			deviceUrl:'',
			euDeviceUrl:'',
			ruDeviceUrl:'',
			height:"100%",
			query_cell_params:{
				group_id:'',
				serial_number:'',
				host_name:'',
				software_version:'',
				search_text:'',
				module_type:'',
				productValue:'',
				rollback_version:"",
				rd: ''
			},
			query_cell_form:{
				group_id:'',
				serial_number:'',
				host_name:'',
				software_version:'',
				search_text:'',
				module_type:'',
				rollback_version:""
			},
			query_eu_params:{
				searchText:'',
				rd: ''
			},
			query_ru_params:{
				searchText:'',
				rd: ''
			},
			productTypeList:[],
			product_type:'',
			list_name:'software',
			list_type:'task',
			seach_device_text:'',
			showTask:true,
			rollbackForm:{
				status:'active',
				exetime:''
			},
			setTimeEnable:true,
			pickerOptions:{
				disabledDate(time){
					return time.getTime()< Date.now()-8.64e7;
				}
			},
			showRollback:false,
			file_type:'upgrade',
			fileImportForm:{
				fileName:''
			},
			showFileImport:false,
			slideUrl:'',
			slideTitle:'',
			slideFooter:'',
			slideHeader:'',
			slidePosition:'',
			slideHeight:'',
			slideWidth:'',
            slideSubmitLoading:'',
			slideModal:'',
			url:'',
			taskUrl:'${ctx}/task/upgrade/getUpgradeTaskList.action',
			taskUrlRb:"",
			dateValue:[],
			query_task_params:{
				timeZone:timeZone,
				taskName:'',
				startTime:'',
				endTime:'',
				searchText:'',
				listType:'upgrade',
				rd: ''
			},
			query_task_form:{
				taskName:'',
				startTime:'',
				endTime:'',
				searchText:''
			},
			list_type_rb:"task",
			showTaskRb:true,
			query_task_params_rb:{
				timeZone:timeZone,
				taskName:'',
				startTime:'',
				endTime:'',
				searchText:'',
				listType:"rollback",
				rd: ''
			},
			query_task_form_rb:{
				taskName:'',
				startTime:'',
				endTime:'',
				searchText:''
			},
			dateValueRb:[],
			file_url:'${ctx}/cell/version/queryfileInfosList.action?file_type=0&isMultiProductType=true',
			query_file_params:{
				timeZone:timeZone,
				searchText:'',
				isMix:true,
				rd: ''
			},
			query_file_form:{
				searchText:''
			},
			menus_file:[],
			rowDataFile:[],
			operType:'',
			menus_task:[],
			menus_device:[],
			rowDataTask:[],
			cellData:[],
			waitNum:"",
			progressNum:"",
			suspendNum:"",
			endNum:"",
			sucNum:"",
			failNum:"",
			resultUrl:"${ctx}/task/upgrade/getUpgradeDeviceList/upgrade.action",
			query_result_params:{
				searchText:"",
				timeZone:timeZone,
				rd: ''
			},
			query_result_params_rb:{
				searchText:"",
				timeZone:timeZone,
				rd: ''
			},
			resultUrlRb:"",
			upgradeTaskAddBtnShow:true,

			advancedQueryItemList:[
				{
					type:'select',
					isShow:true,
					popoverShow:false,
					selectVal:'',
					label:'<%=rb.getString("HuiTuiBanBen") %>',
					options:[],
					value:'rollback_version',
				},
				{
					type:'select',
					isShow:true,
					popoverShow:false,
					selectVal:'',
					label:'<%=rb.getString("RuanJianBanBen") %>',
					options:[],
					value:'software_version',
				},
				{
					type:'select',
					isShow:true,
					popoverShow:false,
					selectVal:'',
					label:'<%=rb.getString("SheBeiZu") %>',
					options:[],
					value:'group_id',
				},
				
			],
			upgradeHeadType:'eNB',
            upgradeHeadBtnData:[],
			importFileShow:false,
			importFileType:'',
			importViewFlag:false,
			importUpgradeFileForm:{
				product:'',
                productList:[],
				file:'',
				fileName:'',
				version:'',
				recommend:'1',
				desc:'',
				to_who:'all',
			},
			importUpgradeFileFormRules:{
				fileName:[
				    {validator: fileNameValidate}
				],
				version:[
				 	{validator: versionValidate,trigger:'blur'}
				],
				product:[
                    {validator: productValidate,trigger:'blur'}
				],
			},
            importCaFileForm:{
				product:'',
				file:'',
				fileName:'',
				version:'',
				recommend:'1',
				desc:'',
				to_who:'all',
			},
			importCaFileFormRules:{
				fileName:[
				 	{validator: fileNameValidate}
				],
				version:[
				 	{validator: versionValidate,trigger:'blur'}
				],
			},
            importFpgaFileForm:{
				product:'',
				file:'',
				fileName:'',
				version:'',
				recommend:'1',
				desc:'',
				to_who:'all',
			},
			importFpgaFileFormRules:{
				fileName:[
				  {validator: fileNameValidate}
				],
				version:[
				 	{validator: versionValidate,trigger:'blur'}
				],
			},
            importApFileForm:{
				product:'',
				file:'',
				fileName:'',
				version:'',
				recommend:'1',
				desc:'',
				to_who:'all',
			},
			importApFileFormRules:{
				fileName:[
				 	{validator: fileNameValidate}
				],
				version:[
				 	{validator: versionValidate,trigger:'blur'}
				],
			},
            upgradeFileType:'',
			showToWho:false,
			fileErrorData:'',
			difFileObj:{
				upgrade:{
					fileTip:'<%=rb.getString("ZhiChiIMGGeShi")%>',
					fileTipYD:'<%=rb.getString("ShengJiZhiChiGeShi")%>',
					fileTip4860:'<%=rb.getString("ZhiZhiChiIMGHeEXTGeShi")%>',
					fileFmt:'IMG',
					fileFmtYD:'tar.gz',
					fileFmt4860:'IMG,EXT',
					fileErrorMsg:'<%=rb.getString("ZhiZhiChiIMGWenJian")%>',
					fileErrorMsgYD:'<%=rb.getString("ShengJiZhiZhiChiWenJian")%>',
					fileErrorMsg4860:'<%=rb.getString("ZhiChiIMGHeEXTGeShi")%>'
				},
				ca:{
					fileTip:'<%=rb.getString("ZhiChiPatchGeShi")%>',
					fileFmt:'patch',
					fileErrorMsg:'<%=rb.getString("ZhiZhiChiPATCHWenJian")%>'
				},
				fpga:{
					fileTip:'<%=rb.getString("ZhiChiIMGGeShi")%>',
					fileFmt:'IMG',
					fileErrorMsg:'<%=rb.getString("ZhiZhiChiIMGWenJian")%>'
				},
				ap:{
					fileTip:'<%=rb.getString("ZhiChiIMGGeShi")%>',
					fileFmt:'IMG',
					fileErrorMsg:'<%=rb.getString("ZhiZhiChiIMGWenJian")%>'
				}
			},
            importFileRefCodes:{
                'upgrade': 'importUpgradeFile',
                'ca': 'importCaFile',
                'fpga': 'importFpgaFile',
                'ap': 'importApFile',
            },
            importFileFormCodes:{
                'upgrade': 'importUpgradeFileForm',
                'ca': 'importCaFileForm',
                'fpga': 'importFpgaFileForm',
                'ap': 'importApFileForm',
            },
            multipartMaxFileSize: multipartMaxFileSize
		}
	},
	methods:{
		init(){
			var vm = this,
				upgradeHeadBtnData = [],
				codes = [
					{label:'eNB',icon:'el-icon el-icon-menu-eNB',code:'CODE_ENB_MONITOR'},
					{label:'gNB',icon:'el-icon el-icon-menu-5G',code:'CODE_GNB_MONITOR'},
					{label:'CPE',icon:'el-icon el-icon-menu-CPE',code:'CODE_CPE_MONITOR'},
					{label:'eGW',icon:'el-icon el-icon-menu-eGW',code:'CODE_EGW'},
				];
			codes.map((item)=>{
				if(writableMap[item.code] != undefined){
					upgradeHeadBtnData.push(item)
				}
			})
			vm.upgradeHeadBtnData = upgradeHeadBtnData;
			if(writableMap["CODE_ENB_MONITOR"] != undefined){
				vm.upgradeHeadType = 'eNB';
				if(isJumpToPage){
					if(isJumpToPage.type == 'view'){
						vm.activeName = 'file';
						if(isJumpToPage.file_type == "0") vm.file_type = "upgrade";
						if(isJumpToPage.file_type == "1") vm.file_type = "ca";
						if(isJumpToPage.file_type == "6") vm.file_type = "fpga";
						vm.changeFileType(vm.file_type,'');
						setTimeout(function(){
							isJumpToPage = '';
						},10)
					}else if(isJumpToPage.type == 'upgrade'){
						vm.addUpgradeTask();
					}
				}
				if(writableMap["CODE_ENB_UPGRADE_IMAGE"] != undefined || writableMap["CODE_ENB_UPGRADE_PATCH"] != undefined || writableMap["CODE_ENB_UPGRADE_FPGA"] != undefined){
					this.list_name = "software"
				}else if(writableMap["CODE_ENB_ROLLBACK"] != undefined){
					this.list_name = "rollback"
				}
				if(writableMap["CODE_ENB_UPGRADE_IMAGE"] ==false && writableMap["CODE_ENB_UPGRADE_PATCH"] == false && writableMap["CODE_ENB_UPGRADE_FPGA"] == false){
					this.upgradeTaskAddBtnShow = false
				}else{
					this.upgradeTaskAddBtnShow = true
				}
				axios.post('${ctx}/task/upgrade/getProductType.action').then(function(response){
                    var data = response.data || [];
                    data.map((item)=>{
                        if(item.name == 'DXDF'){
                            item.fileType = 'tar.gz';
                        }else if(item.name == 'CR-B4860/EU' || item.name == 'CR-B4860/RU'){
                            item.fileType = 'IMG,EXT';
                        }else{
                            item.fileType = 'IMG';
                        }
                    })
					vm.productTypeList = data;
					vm.$nextTick(function(){
						vm.product_type = vm.productTypeList[0].value;
						var reg = new RegExp('\\\\',"g");
						if(vm.product_type != 'CR-B4860/EU' && vm.product_type != 'CR-B4860/RU'){
							vm.query_cell_params.productValue = vm.productTypeList[0].value.includes("CR-B4860") ? vm.productTypeList[0].value.substr(0,8) : vm.productTypeList[0].value.replace(reg,'');
						}
						
						vm.deviceUrl = '${ctx}/task/upgrade/queryCellInfos.action?forSelect=1';
						vm.euDeviceUrl = '${ctx}/cell/nxp/queryAllEUInfos.action';
						vm.ruDeviceUrl = '${ctx}/cell/nxp/queryAllRUInfos.action';
					});
				}).catch(function(error){})

				if(isCloudCore == 'true'){
					vm.showToWho = true;
				}else{
					vm.showToWho = false;
				}
				setTimeout(function(){
					vm.commonSelection();
				},500)
			}else if(writableMap["CODE_GNB_MONITOR"] != undefined){
				vm.upgradeHeadType = 'gNB';
				$("#gnbUpgrade").load('${ctx}/gnb/upgrade/toGnodeBFileListPage.action?omcVersion=&menu_id=21004',function(data){
					$.parser.parse(this);
				});
				
			}else if(writableMap["CODE_CPE_MONITOR"] != undefined){
				vm.upgradeHeadType = 'CPE';
				$("#cpeUpgrade").load('${ctx}/cell/version/toUploadFile.action?cpe=1&omcVersion=&menu_id=7004',function(data){
					$.parser.parse(this);
				});
				
			}else if(writableMap["CODE_EGW"] != undefined){
				vm.upgradeHeadType = 'eGw';
				$("#egwUpgrade").load('${ctx}/egw/pageForward/goEGWUpgrade.action?omcVersion=&menu_id=10009',function(data){
					$.parser.parse(this);
				});
			}		
			
				
		},
		upgradeHeadBtnClick(type){
			var vm = this;

			if(type == 'eNB'){

			}else if(type == 'gNB'){
				$("#gnbUpgrade").load('${ctx}/gnb/upgrade/toGnodeBFileListPage.action?omcVersion=&menu_id=21004',function(data){
					$.parser.parse(this);
				});
			}else if(type == 'CPE'){
				$("#cpeUpgrade").load('${ctx}/cell/version/toUploadFile.action?cpe=1&omcVersion=&menu_id=7004',function(data){
					$.parser.parse(this);
				});
			}else if(type == 'eGW'){
				$("#egwUpgrade").load('${ctx}/egw/pageForward/goEGWUpgrade.action?omcVersion=&menu_id=10009',function(data){
					$.parser.parse(this);
				});
			}	
		},
		query(val){
			this.query_cell_params.search_text = val;
			this.query_cell_params.rd = Math.random();
		},
		queryEU(val){
			this.query_eu_params.search_text = val;
			this.query_eu_params.rd = Math.random();
		},
		queryRU(val){
			this.query_ru_params.search_text = val;
			this.query_ru_params.rd = Math.random();
		},
		setTime(){
			this.rollbackForm.exetime = formatDate(new Date(gloableTime));
			this.$refs.rollbackForm.validateField('exetime');
		},
		addUpgradeTask(){
			var vm = this;
			vm.slideHeader = true;
    	    vm.slideTitle = '<%=rb.getString("XinJianRenWu")%>';
    	    vm.slideUrl = "${ctx}/task/upgrade/goAddTask.action";
    	    vm.slideFooter = 'true';
    	    vm.slidePosition = 'top';
    	    vm.slideHeight = '100%';
    	    vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
    	    vm.operType = 'addTask';
    	    vm.$refs.slide.showSlide(function(){
    	    	vm.modal = false;
    	    });
		},
		cancelSlide(){
			if(this.operType == "addTask" ||　this.operType == "modifyTask"){
				eventBus.$emit('cancel-add-task');
			}else if(this.operType == "importFile"){
				eventBus.$emit('cancel-import');
			}else if(this.operType == "editFile"){
				eventBus.$emit('cancel-edit');
			}else{
				this.$refs.slide.hide();
			}
		},
		saveSlide(){
			if(this.operType == "addTask" ||　this.operType == "modifyTask"){
				eventBus.$emit('add-task');
			}
			if(this.operType == "importFile"){
				eventBus.$emit('import-file');
			}
			if(this.operType == "editFile"){
				eventBus.$emit('edit-file');
			}
		},
		//改变产品类型
		changeProduct(val){
			var vm = this,
				reg = new RegExp('\\\\',"g"),
				productIntVal = val;
			
			vm.product_type = val;
			vm.cellData = [];
			if(productIntVal != 'CR-B4860/EU' && productIntVal != 'CR-B4860/RU'){
				if(val.includes("CR-B4860")){
					var index = val.indexOf("/");
					val = val.substring(0,index)
				}
				vm.euDeviceUrl = '';
				vm.ruDeviceUrl = '';
				vm.$refs.upgrade_cell_table.clearSelection();
				vm.query_cell_params.productValue = val.replace(reg,'');
			}else{
				vm.$refs.upgrade_ru_table.clearSelection();
				vm.$refs.upgrade_eu_table.clearSelection();
				if(productIntVal == 'CR-B4860/EU'){
					vm.ruDeviceUrl = '';
					vm.euDeviceUrl = '${ctx}/cell/nxp/queryAllEUInfos.action'
				}else{
					vm.euDeviceUrl = '';
					vm.ruDeviceUrl = '${ctx}/cell/nxp/queryAllRUInfos.action'
				}
			}
			setTimeout(function(){
				vm.commonSelection();
			},500)
		},		
		commonSelection(){
			var vm = this;
			var reg = new RegExp('\\\\',"g");
			var productVal = vm.product_type.includes("CR-B4860") ? vm.product_type.substr(0,8) : vm.product_type.replace(reg,'');
			
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : productVal,
				selectType : 'deviceGroup'
			})).then(function(response){
				var data = response.data,
					groupOptionsList=[];

				data.map((item)=>{
					groupOptionsList.push({label:item.text,value:item.value})
				})
				vm.advancedQueryItemList.map((items)=>{
					if('group_id' == items.value){
						items.options = groupOptionsList
					}
				})
			}).catch(function(error){})
			
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : productVal,
				selectType : 'version'
			})).then(function(response){
				var data = response.data,
					versionOptionsList=[];

				data.map((item)=>{
					versionOptionsList.push({label:item.text,value:item.value})
				})
				vm.advancedQueryItemList.map((items)=>{
					if('software_version' == items.value){
						items.options = versionOptionsList
					}
				})
			}).catch(function(error){})
			
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : productVal,
				selectType : 'rollbackVersion'
			})).then(function(response){
				var data = response.data,
					rollbackVersionList=[];

				data.map((item)=>{
					rollbackVersionList.push({label:item.text,value:item.value})
				})
				vm.advancedQueryItemList.map((items)=>{
					if('rollback_version' == items.value){
						items.options = rollbackVersionList
					}
				})
			}).catch(function(error){})			
		},
		queryTask(val){
			if(this.list_name == "software"){
				this.query_task_params.searchText = val;
				this.query_task_params.rd = Math.random();
			}else if(this.list_name == "rollback"){
				this.query_task_params_rb.searchText = val;
				this.query_task_params_rb.rd = Math.random();
			}
		},
		resultFmt(row,column,cellValue,index){//软件升级 任务结果fmt
	    	var resultObj = {
					"1" : "<%=rb.getString("ChengGong")%>",
					"2" : "<%=rb.getString("BuFenChengGong")%>",
					"3" : "<%=rb.getString("ShiBai")%>",
					"" : ""
			}
			return resultObj[cellValue];
	    },
	    deviceResultFmt(row,column,value,index) {
	    	if (value == "3") {
				return ShiBai;
			} else if (value == "1") {
				return ChengGong;
			} else if (value == "2") {
				return ZhongZhi;
			}else {
				return "";
			}
	    },
	    changeListType(val){
	    	this.showTask = val=='task'?true:false;
	    },
	    changeListRb(val){
	    	this.showTaskRb = val=='task'?true:false;
	    },
	    queryFile(){
	    	Object.assign(this.query_file_params,this.query_file_form);
	    	this.query_file_params.rd = Math.random();
	    },
	    changeFileType(newVal){
            var vm = this,
                urlObj = {
	    			"upgrade" : "${ctx}/cell/version/queryfileInfosList.action?file_type=0&isMultiProductType=true",
	    			"ca" : "${ctx}/cell/version/queryfileInfosList.action?file_type=1",
	    			"fpga" : "${ctx}/cell/version/queryfileInfosList.action?file_type=6",
	    			"ap" : "${ctx}/cell/version/queryfileInfosList.action?file_type=11"
	    	    };
            vm.file_url = "";
	    	vm.query_file_form.searchText = "";
	    	vm.query_file_params.searchText = "";
	    	vm.$nextTick(function(){
	    		vm.file_url = urlObj[newVal];
	    	})
            if(vm.importFileShow){
                vm.rightBoxClose();
            }
	    },
	    importFileClick(){
	    	var vm = this;
			
			vm.importFileType = 'add';
			vm.importViewFlag = false;
			vm.$refs[vm.importFileFormCodes[vm.file_type]].resetFields();
			vm.importFileShow = true;
	    },
	    cancalImportFile(){
	    	this.$refs.tslide.hide();
	    },
	    optClickTask(row,ev){
	    	var vm = this;
   	    	var status = row.TASK_STATUS;
   		    vm.rowDataTask = row;
   		    var type = row.TYPE;
   		    var codeObj = {
   		    		"1" : "CODE_ENB_UPGRADE_IMAGE hidden",
   		    		"4" : "CODE_ENB_UPGRADE_PATCH hidden",
   		    		"6" : "CODE_ENB_UPGRADE_FPGA hidden"
   		    }
   		    var codeName;
   		    if(vm.list_name == "software"){
   		    	codeName = codeObj[type];
   		    }else{
   		    	codeName = "CODE_ENB_ROLLBACK hidden"
   		    }
	    	vm.menus_task= [
		          {label:'<%=rb.getString("KaiShi")%>',cls:codeName,code:'start'},
		          {label:'<%=rb.getString("ZanTing")%>',cls:codeName,code:'stop'},
		          {label:'<%=rb.getString("ZhongZhiRenWu")%>',cls:codeName,code:'terminate'},
		          {label:'<%=rb.getString("XinXi")%>',code:'view'},
		          {label:'<%=rb.getString("XiuGai")%>',cls:codeName,code:'edit'},
		          {label:'<%=rb.getString("ShanChu")%>',cls:codeName,code:'del'},
		    ]
	    	initTaskStatus(status,vm.menus_task);
	    	vm.$nextTick(function(){
	    		document.body.click();
   		    	vm.$refs.menu_task.show(ev);
	    	});
	    },
	    clickMenuTask(ev){
	    	var codes = {
   	    		view:this.viewTask,
   	    		start:this.activeTask,
   	    		stop:this.suspendTask,
   	    		terminate:this.terminateTask,
   	    		edit:this.editTask,
   	    		del:this.delTask,
    	    }
   	    	if(codes[ev.code]){
   	    		codes[ev.code](this.rowDataTask["TASK_ID"],this.rowDataTask["TYPE"])
   	    	}
	    },
	    activeTask(task_id,type){ // 开始
	    	var vm = this;
	    	axios.post('${ctx}/task/upgrade/activeTask.action',stringify({
	    		taskId : task_id,
	    		type : type
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			if(vm.list_name == "software"){
	    				vm.$refs.upgrade_task_table.refresh()
	    			}else{
	    				vm.$refs.rb_task_table.refresh()
	    			}
	    		}else{
	    			vm.$message.error(data["message"])
	    		}
	    	})
	    },
	    suspendTask(task_id,type){ // 暂停
	    	var vm = this;
	    	axios.post('${ctx}/task/upgrade/suspendTask.action',stringify({
	    		taskId : task_id,
	    		type : type
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			if(vm.list_name == "software"){
	    				vm.$refs.upgrade_task_table.refresh()
	    			}else{
	    				vm.$refs.rb_task_table.refresh()
	    			}
	    		}else{
	    			vm.$message.error(data["message"])
	    		}
	    	})
	    },
	    terminateTask(task_id,type){ // 终止任务
	    	var vm = this;
	    	axios.post('${ctx}/task/upgrade/terminateUpgradeTask.action',stringify({
	    		taskId : task_id,
	    		type : type
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			if(vm.list_name == "software"){
	    				vm.$refs.upgrade_task_table.refresh()
	    			}else{
	    				vm.$refs.rb_task_table.refresh()
	    			}
	    		}else{
	    			vm.$message.error(data["message"])
	    		}
	    	})
	    },
        viewTask(task_id,type){ //  信息查看
        	var vm = this;
            vm.slideTitle = '<%=rb.getString("XinXi")%>';
            vm.operType = 'viewTask';
            vm.slideFooter = false;
            if(vm.list_name == "software"){
	    		vm.slideUrl = "${ctx}/task/upgrade/goAddTask.action";
	    	}else{
	    		vm.slideUrl = "${ctx}/task/upgradeRollback/goAddTask.action";
	    	}
	    	vm.slideHeader = true;
    	    vm.slidePosition = 'top';
    	    vm.slideHeight = '100%';
    	    vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
    	    vm.$refs.slide.showSlide(function(){
    	    	vm.modal = false
    	    });
        },
	    editTask(task_id,type){ //  修改任务
	    	var vm = this;
	    	vm.slideTitle = '<%=rb.getString("XiuGai")%>';
            vm.operType = 'modifyTask';
            vm.slideFooter = true;
	    	if(vm.list_name == "software"){
	    		vm.slideUrl = "${ctx}/task/upgrade/goAddTask.action";
	    	}else{
	    		vm.slideUrl = "${ctx}/task/upgradeRollback/goAddTask.action";
	    	}
	    	vm.slideHeader = true;
    	    vm.slidePosition = 'top';
    	    vm.slideHeight = '100%';
    	    vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
    	    vm.$refs.slide.showSlide(function(){
    	    	vm.modal = false
    	    });
	    },
	    delTask(task_id,type){//软件升级  删除任务
	    	var vm = this;
	    	this.$confirm('<%=rb.getString("QueRenShanChuRenWu")%>',QueRen,{
	    		customClass:'warningConfirm',
	    		confirmButtonText:'<%=rb.getString("QueDing")%>',
	    		cancelButtonText:'<%=rb.getString("QuXiao")%>',
	    		type:'warning',
	    		closeOnClickModal:false
	    	}).then(() => {
	    		axios.post('${ctx}/task/upgrade/delUpgradeTask.action',stringify({
		    		taskId:task_id,
		    		type:type
		    	})).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
		    			if(vm.list_name == "software"){
		    				vm.$refs.upgrade_task_table.refresh()
		    			}else{
		    				vm.$refs.rb_task_table.refresh()
		    			}
		    			vm.$message({
			    			type:'success',
			    			message:'<%=rb.getString("ChengGong")%>'
			    		})
		    		}else{
		    			vm.$message.error(data["message"])
		    		}
		    	}).catch(function(error){
		    		
		    	})
	    	}).catch()
	    },
	    retryTask(task_id){
			var vm = this;
			var params = {
					taskId : task_id
			}
			axios.post("${ctx}/task/upgrade/reTryUpgrade.action",stringify(params)).then(function(response){
				var data = response.data;
				if(data) {
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success'
						});
						vm.$refs.upgrade_task_table.refresh()
					}else{
						vm.$message.error(data["message"])
					}
				}
			})
		},
		optClickFile(row,ev){
			var vm = this;
			vm.rowDataFile = row;
			var TuiJian = '',
				recommendIcon = '',
				modifyShow = false,
				deleteShow = true;
			var activeName = vm.file_type;
			var showCls = {
    			'upgrade':'CODE_ENB_UPGRADE_FILE hidden',
    			'ca':'CODE_ENB_UPGRADE_FILE hidden',
    			'fpga':'CODE_ENB_UPGRADE_FILE hidden',
                'ap':'CODE_ENB_UPGRADE_FILE hidden',
    		}
			if(row.recommend == '1'){//说明此文件是推荐文件
				TuiJian = '<%=rb.getString("QuXiaoTuiJian")%>';
				recommendIcon = 'el-icon el-icon-operation-cancel-recommend';
			}else{
				TuiJian = '<%=rb.getString("TuiJian")%>';
				recommendIcon = 'el-icon el-icon-operation-recommend';
			}
			if( is_super_user == 'true' ){
				modifyShow = true;
			}else{
				deleteShow = false;
			}
			vm.menus_file = [
				{label:"<%=rb.getString("XinXi")%>",cls:"el-icon el-icon-operation-info",code:"view"},
				{label:"<%=rb.getString("XiaZai")%>",cls:"el-icon el-icon-operation-download" + " " +showCls[activeName],code:"download"},
				{label:"<%=rb.getString("XiuGai")%>",cls:"el-icon el-icon-operation-edit" + " " +showCls[activeName],code:"modify"},
				{label:"<%=rb.getString("ShanChu")%>",cls:"el-icon el-icon-operation-delete" + " " +showCls[activeName],code:"del"},
				{label:TuiJian,cls:recommendIcon + " " +showCls[activeName],code:"recommend",show: modifyShow}
			]
			vm.$nextTick(function(){
				document.body.click();
				vm.$refs.menu_file.show(ev);
			})
		},
		clickMenuFile(ev){
			var codes = {
					view:this.viewFile,
					modify:this.editFile,
					download:this.downloadFile,
					del:this.delFile,
					recommend:this.recommendFile
			}
			if(codes[ev.code]){
				codes[ev.code](this.rowDataFile)
			}
		},
		handerClose(){
			this.$refs.menu_task.hide();
			this.$refs.menu_file.hide();
		},
		viewFile(row){
			var vm = this;
			
			vm.importFileType = 'view';
			vm.importViewFlag = true;
			Object.keys(vm[vm.importFileFormCodes[vm.file_type]]).forEach(function(key){
				if(key == 'fileName'){
					vm[vm.importFileFormCodes[vm.file_type]][key] = row.file_name ? row.file_name : '';
				}else if(key == 'product'){
					vm[vm.importFileFormCodes[vm.file_type]][key] = row.productValue ? row.productValue : '';
                    if(vm.file_type == 'upgrade'){
                        vm[vm.importFileFormCodes[vm.file_type]]['productList'] = row.productValue ? row.productValue.split(',') : [];
                    }
				}else if(key == 'to_who') {
					vm[vm.importFileFormCodes[vm.file_type]][key] = row.toWho ? row.toWho : '';
				}else{
					if(row[key] != undefined && row[key] != null){
						vm[vm.importFileFormCodes[vm.file_type]][key] = row[key];
					}
				}
			});
			vm.importFileShow = true;
		},
		editFile(row){
			var vm = this;
			
			vm.importFileType = 'edit';
			vm.importViewFlag = false;
			Object.keys(vm[vm.importFileFormCodes[vm.file_type]]).forEach(function(key){
				if(key == 'fileName'){
					vm[vm.importFileFormCodes[vm.file_type]][key] = row.file_name ? row.file_name : '';
				}else if(key == 'product'){
					vm[vm.importFileFormCodes[vm.file_type]][key] = row.productValue ? row.productValue : '';
                    if(vm.file_type == 'upgrade'){
                        vm[vm.importFileFormCodes[vm.file_type]]['productList'] = row.productValue ? row.productValue.split(',') : [];
                        let productStr = row.productValue ? row.productValue : '';
                        let isExistDxdf = false,
                            isExist4860 = false,
                            isExistOther = false;
                        vm.productTypeList.map(item => {
                            if(productStr.includes(item.value)){
                                if(item.fileType == 'tar.gz'){
                                    isExistDxdf = true;
                                }else if(item.fileType == 'IMG,EXT'){
                                    isExist4860 = true;
                                }else{
                                    isExistOther = true;
                                }
                            }
                        });
                        if(isExistDxdf && !isExist4860 && !isExistOther){
                            vm.upgradeFileType = 'tar.gz';
                        }else if(!isExistDxdf && isExist4860 && !isExistOther){
                            vm.upgradeFileType = 'IMG,EXT';
                        }else if(!isExistDxdf && !isExist4860 && isExistOther || !isExistDxdf && isExist4860 && isExistOther){
                            vm.upgradeFileType = 'IMG';
                        }else{
                            vm.upgradeFileType = '';
                        }
                    }
				}else if(key == 'to_who') {
					vm[vm.importFileFormCodes[vm.file_type]][key] = row.toWho ? row.toWho : '';
				}else{
					if(row[key] != undefined && row[key] != null){
						vm[vm.importFileFormCodes[vm.file_type]][key] = row[key];
					}
				}
			});
			vm.importFileShow = true;

		},
		downloadFile(){
			var vm = this;
			var codes = {
					upgrade : 1,
					ca : 3,
					fpga : 6,
					ap : 11
			}
			var params = {
					fileName : vm.rowDataFile.file_name,
					fileType : codes[vm.file_type]
			}
			axios.post("${ctx}/omc/version/file/fileIsExist.action",stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
					var url = '${ctx}/omc/version/file/downLoadFile.action';
					exportByForm(url,params);
				}else{
					vm.$message.error(data["message"])
				}
			})
		},
		delFile(){
			var vm = this;
			var confirmStr = '<%=rb.getString("QueRenShanChu")%>';
			var url = '${ctx}/cell/version/deleteVersionFile.action';
			var params = {
					fileID :vm.rowDataFile.id
			}
			vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
				customClass:"warningConfirm",
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(() => {
				axios.post(url,stringify(params)).then(function(response){
					var data = response.data;
					if(data["success"]){
						vm.$message({
    						message:"<%=rb.getString("ChengGong")%>",
    						type:'success',
    					})
                        vm.$refs.file_table.refresh();
					}else{
						vm.$message.error(data["message"])
					}
				}).catch(() => {})
			})
		},
		recommendFile(){
			var vm = this;
			var params = {
					id : vm.rowDataFile.id
			}
			if(vm.rowDataFile.recommend == '0'){
				params.recommend = '1'
			}else{
				params.recommend = '0'
			}
			axios.post("${ctx}/cell/version/updateRecommendStatus.action",stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
					vm.$refs.file_table.refresh();
				}else{
					vm.$messager.error(data["message"]);
				}
			})
		},
		clickMenuDevice(){
			
		},
		selectCell(selection){
			this.cellData = selection;
		},
		addRbTask(){
			var vm = this;
			vm.slideTitle = '<%=rb.getString("ShengJiHuiTuiRenWu")%>';
    	    vm.slideUrl = "${ctx}/task/upgradeRollback/goAddTask.action";
    	    vm.slideFooter = true;
    	    vm.slideHeader = true;
    	    vm.slidePosition = 'top';
    	    vm.slideHeight = '100%';
    	    vm.slideWidth = '100%';
            vm.slideSubmitLoading = false;
    	    vm.operType = 'addTask';
    	    vm.$refs.slide.showSlide(function(){
    	    	vm.modal = false
    	    });
		},
		typeFmt(row,column,cellValue,index){//软件升级 任务结果fmt
	    	var typeObj = {
					"1" : "Software Upgrade",
					"4" : "PATCH Upgrade",
					"6" : "FPGA Upgrade",
					"7" : "AP Upgrade",
					"" : ""
			}
			return typeObj[cellValue];
	    },
	    loadSuccessTask(data){
	    	this.waitNum = data.properties.watingCount;
	    	this.progressNum = data.properties.inProgressCount;
	    	this.suspendNum = data.properties.suspendCount;
	    	this.endNum = data.properties.endCount;
	    },
	    loadsuccessResult(data){
	    	this.sucNum = data.properties.successCount;
	    	this.failNum = data.properties.failureCount;
	    },
	    queryResult(val){
	    	if(this.list_name == "software"){
				this.query_result_params.searchText = val;
	    		this.query_result_params.rd = Math.random();
	    	}else{
				this.query_result_params_rb.searchText = val;
	    		this.query_result_params_rb.rd = Math.random();
	    	}
	    	
	    },
	    clickList(tab){
	    	if(tab.name == "software"){
	    		this.$refs.upgrade_task_table.refresh();
	    		this.$refs.upgrade_result_table.refresh();
	    		this.taskUrlRb = "";
	    		this.resultUrlRb = "";
	    	}else{
	    		this.taskUrlRb = "${ctx}/task/upgrade/getUpgradeTaskList.action";
	    		this.resultUrlRb = "${ctx}/task/upgrade/getUpgradeDeviceList/rollback.action";
	    	}
	    },
	    exportResult(){
	    	var vm = this;
	    	var url = "";
	    	var params = {}
	    	if(vm.list_name == "software"){
	    		url = "${ctx}/task/upgrade/exportUpgradeDeviceListToCSV/upgrade.action";
	    		params.timeZone = timeZone;
	    		params.searchText = vm.query_result_params.searchText;
	    	}else if(vm.list_name == "rollback"){
	    		url = "${ctx}/task/upgrade/exportUpgradeDeviceListToCSV/rollback.action";
	    		params.timeZone = timeZone;
	    		params.searchText = vm.query_result_params_rb.searchText;
	    	}
	    	exportByForm(url,params);
	    },
	    towhoFmt(row,column,cellVal,index){
			if(cellVal =='all'){
				return "GA"
			}else if(cellVal == 'beta'){
				return "Beta"
			}else if(cellVal == 'none'){
				return "Test"
			}
		},
		configFmt(row,column,cellValue,index){
	    	if(cellValue == 'true'){
	    		return "<%=rb.getString("Fou")%>"
	    	}else{
	    		return "<%=rb.getString("Shi")%>"
	    	}
	    },
	    restartTask(type,taskId,snCode){
	    	var vm = this;
	    	var result = [];
	    	if(type == 'single'){
	    		result = [{[taskId]:snCode}]
	    	}
	    	var params = {
	    			taskDeviceList : JSON.stringify(result)
	    	}
			axios.post("${ctx}/task/upgrade/reTryUpgrade.action",stringify(params)).then(function(response){
				var data = response.data;
				if(data) {
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success'
						});
						vm.$refs.upgrade_task_table.refresh()
						vm.$refs.upgrade_result_table.refresh()
					}else{
						vm.$message.error(data["message"])
					}
				}
			})
	    },
	    dateChange(val) {
			var vm = this;
			vm.dateValue = val;
			if(val != null){
				vm.query_task_params.startTime = vm.dateValue[0];
				vm.query_task_params.endTime = vm.dateValue[1];
			}
		},
		dateChangeSoft(val) {
			var vm = this;
			vm.dateValueRb = val;
			if(val != null){
				vm.query_task_params_rb.startTime = vm.dateValueRb[0];
				vm.query_task_params_rb.endTime = vm.dateValueRb[1];
			}
		},
		// 高级查询 确定事件
		advanceQuery(type,paramsItem,value){
			var vm = this,
				params ={};
			if(type == 'select'){
				params[paramsItem] = value;
			}else{
				params[paramsItem] = value.join(',');
			}
			Object.assign(vm.query_cell_params, params);
		},
		// 清除筛选
		clearFilterClick(){
			var vm = this,
				params = {};
			vm.advancedQueryItemList.map((items)=>{
				if(items.isShow && items.isShow== true ){
					items.selectVal = '';
					params[items.value] = '';
				}
			})
			Object.assign(vm.query_cell_params, params);
			document.body.click();
		},
		// 升级文件选择
		importFileChange(file, fileList) {
			var vm = this,
				fileSize = file.size;

            if(fileSize <= vm.multipartMaxFileSize){
                vm.fileErrorData = '';
                vm[vm.importFileFormCodes[vm.file_type]].file = file.raw;
                vm[vm.importFileFormCodes[vm.file_type]].fileName = file.name;
                if(vm.importFileType == 'add'){
                    vm.getVersion(file.name);
                }
            }else{
                var maxFileSizeMB = Number(vm.multipartMaxFileSize / 1024 / 1024).toFixed(0),
                    fileSizeMB = Number(fileSize / 1024 / 1024).toFixed(0),
                    messageStr = '<%=rb.getString("CollectLogSizeExceedOne")%>' + fileSizeMB + '<%=rb.getString("CollectLogSizeExceedTwo")%>' + maxFileSizeMB + '<%=rb.getString("CollectLogSizeExceedThree")%>';
                vm.$message({
                    type: 'error',
                    message: messageStr
                });

                vm.$refs[vm.importFileRefCodes[vm.file_type]].clearFiles();
            }
		},
		/**
		* 文件上传之前
		* @param file{object}   文件信息
		*/ 
		beforeUpload(file){
			var vm = this;
			var fileName = file.name,fileSize = file.size;
			var fd = new FormData(),
				config = {
					headers: { 'Content-Type': 'multipart/form-data' },
					onUploadProgress:(ev)=>{
						if(ev.lengthComputable || ev.event.lengthComputable) {
							var total = ev.total,
								loaded = ev.loaded,
								percent = 100*loaded/total;
							$('#progressUploadFile').progressbar('setValue', percent.toFixed(2));
						}
					}
				};
			fd.append('uploadFile',file); //文件流
			fd.append('newFileName',fileName); //文件流
			fd.append('fileSize',fileSize);//文件大小
			fd.append('desc',vm[vm.importFileFormCodes[vm.file_type]].desc);//描述
			fd.append('deviceType','');
			fd.append('md5','');
			// fd.append('token',omctoken);
			if(vm.file_type == "ap"){
				fd.append('product','ap');
			}else{
                let productStr = vm[vm.importFileFormCodes[vm.file_type]].product;
				fd.append('product',productStr);
			}
			
			fd.append('version',vm[vm.importFileFormCodes[vm.file_type]].version);
			fd.append('recommend',vm[vm.importFileFormCodes[vm.file_type]].recommend);
			if(vm.showToWho){
				fd.append('to_who',vm[vm.importFileFormCodes[vm.file_type]].to_who);
			}else{
				fd.append('to_who','all');
			}
            fd.append('fileType',vm.file_type);
			vm.fileErrorData = vm.$refs[vm.importFileRefCodes[vm.file_type]].uploadFiles[0];
			$('#progressUploadFile').progressbar('setValue', 0);// 将进度条进度置为0
			$("#winUploadPro").window("open");// 打开进度条窗口
			axios.post("${ctx}/cell/version/uploadVersionFile.action",fd,config).then(function(response){
				var data = response.data
				$("#winUploadPro").window("close");// 关闭进度条窗口
				if(data["MD5"]){
					vm.fileErrorData = '';
					$.messager.alert('<%=rb.getString("TiShi")%>','<%=rb.getString("ShangChuanChengGong")%><%=rb.getString("DouHao")%><%=rb.getString("WenJianMD5Zhi")%><%=rb.getString("MaoHao")%>'+data["MD5"]);
					vm.$refs.file_table.refresh();
					vm.rightBoxClose();
				}else{
					vm.$message.error(data["message"]);
				}
			})
			
			return false;
		},
		//发送请求，校验device文件内容 
		checkImportFile(res, file) {    
			var vm = this;
			if (res.success) {
				if (res.suc_count > 0) {
					vm.$message({
						type: 'success',
						message: '<%=rb.getString("ChengGong")%>'
					});
				} else {
					vm.$message({
						type: 'warning',
						message: '<%=rb.getString("ShiBai")%>'
					});
				}
			} else {
				vm.$message({
					type: 'error',
					message: res.msg
				});
			}
			//修改已选择文件状态  
			var fileList = vm.$refs[vm.importFileRefCodes[vm.file_type]].uploadFiles;
			fileList.forEach(function (file) {
				file.status = 'ready';
			})
		},
		importFileSelect() { 
			var vm = this;
			vm.$refs[vm.importFileRefCodes[vm.file_type]].clearFiles();
			vm.$refs['file_up_'+ vm.file_type].click();
		},
		getVersion(val){
			var vm = this,
                fileFmt = '';
            if(vm.file_type == 'upgrade'){
                if(vm.upgradeFileType == 'tar.gz'){
                    fileFmt = vm.difFileObj[vm.file_type].fileFmtYD;
                }else if(vm.upgradeFileType == 'IMG,EXT'){
                    fileFmt = vm.difFileObj[vm.file_type].fileFmt4860;
                }else if(vm.upgradeFileType == 'IMG'){
                    fileFmt = vm.difFileObj[vm.file_type].fileFmt;
                }
            }else{
                fileFmt = vm.difFileObj[vm.file_type].fileFmt;
            }
			if(val != '' && fileFmt != '' && vm.fileFormatMatch(val,fileFmt)){
                var pathSplit = val.split(/\\/);
                var filename = pathSplit[pathSplit.length - 1];
			    if(filename.substring(filename.length-6) == 'tar.gz'){
			 		vm[vm.importFileFormCodes[vm.file_type]].version = filename.substring(0,filename.length-7);
			 	}else{
			 		vm[vm.importFileFormCodes[vm.file_type]].version = filename.substring(0,filename.lastIndexOf("."));
			 	}
			 	vm.$refs[vm.importFileFormCodes[vm.file_type]].validateField('version')
			 }
		},
		// 文件校验
		fileFormatMatch(str,regs){
			var regsArr = regs.toLowerCase().split(",");
			if(str.substring(str.length-6) == 'tar.gz'){
				var suffix = "tar.gz";
			}else{
				var suffix = str.substring(str.lastIndexOf(".")+1).toLowerCase();
			}
			if(regsArr.indexOf(suffix)>-1){
				return true;
			}else{
				return false;
			}
		},
		// 升级文件导入确定
		importFileSubmit(){
			var vm = this;
			vm.$refs[vm.importFileFormCodes[vm.file_type]].validate((valid) => {
				if(valid){
					if(vm.importFileType == 'add'){
						if(vm.fileErrorData){
							vm.$refs[vm.importFileRefCodes[vm.file_type]].uploadFiles.push(vm.fileErrorData);
						}
						vm.$refs[vm.importFileRefCodes[vm.file_type]].submit();
					}else{
						vm.fileImportEditSubmit()
					}
				}
			})
		},
		// 升级文件 修改提交
		fileImportEditSubmit(){
			var vm = this,
				codes = {upgrade:0,ca:1,fpga:6,ap:11},
				params={
					versionId:vm.rowDataFile.id,
					fileName:vm[vm.importFileFormCodes[vm.file_type]].fileName,
					product:vm[vm.importFileFormCodes[vm.file_type]].product,
					version:vm[vm.importFileFormCodes[vm.file_type]].version,
					recommend:vm[vm.importFileFormCodes[vm.file_type]].recommend,
					desc:vm[vm.importFileFormCodes[vm.file_type]].desc,
					toWho : vm[vm.importFileFormCodes[vm.file_type]].to_who
				};
            params.fileType = codes[vm.file_type];
			axios.post("${ctx}/cell/version/goModifyDeviceVersionFileInfo.action",stringify(params)).then(function(response){
				var data = response.data;
				if(data) {
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success'
						});
						vm.$refs.file_table.refresh();
						vm.rightBoxClose();
					}else{
						vm.$message.error(data["message"])
					}
				}
			}).catch(function(error){})
		},
		// 升级文件导入 关闭
		rightBoxClose(){
			var vm = this,
				params = {
					product:'',
					file:'',
					fileName:'',
					version:'',
					recommend:'1',
					desc:'',
					to_who:'all',
				};
            
			vm.importFileShow = false;
            ['upgrade','ca','fpga','ap'].map(item => {
                if(item == 'upgrade'){
                    params.productList = [];
                }
                if(item == 'ap'){
                    delete params.product;
                }
                Object.assign(vm[vm.importFileFormCodes[item]],params)
            });
            vm.upgradeFileType = '';

		},
		fileTypeClick(event){
			var vm = this;
			
			event.preventDefault();
		},
        productListChange(val){
            var vm = this,
                selectList = val || [];
                productStr = selectList.join(',');
            if(vm.file_type == 'upgrade'){
                vm[vm.importFileFormCodes[vm.file_type]].product = productStr;
                let isExistDxdf = false,
                    isExist4860 = false,
                    isExistOther = false;
                vm.productTypeList.map(item => {
                    if(productStr.includes(item.value)){
                        if(item.fileType == 'tar.gz'){
                            isExistDxdf = true;
                        }else if(item.fileType == 'IMG,EXT'){
                            isExist4860 = true;
                        }else{
                            isExistOther = true;
                        }
                    }
                });
                if(isExistDxdf && !isExist4860 && !isExistOther){
                    vm.upgradeFileType = 'tar.gz';
                }else if(!isExistDxdf && isExist4860 && !isExistOther){
                    vm.upgradeFileType = 'IMG,EXT';
                }else if(!isExistDxdf && !isExist4860 && isExistOther || !isExistDxdf && isExist4860 && isExistOther){
                    vm.upgradeFileType = 'IMG';
                }else{
                    vm.upgradeFileType = '';
                }
                if(vm.upgradeFileType && vm[vm.importFileFormCodes[vm.file_type]].fileName != ''){
                    vm.getVersion(vm[vm.importFileFormCodes[vm.file_type]].fileName);
                }
            }else{
                if(vm[vm.importFileFormCodes[vm.file_type]].fileName != ''){
                    vm.getVersion(vm[vm.importFileFormCodes[vm.file_type]].fileName);
                }
            }
            vm.$refs[vm.importFileFormCodes[vm.file_type]].validateField('product');
        }
	},
	computed: {
		rightOutBoxTitle() {
			var vm = this,
				codes={
					'add':'File Import',
					'view':'File Information',
					'edit':'File Modify'
				};

			return codes[this.importFileType];
		}
	},
	watch:{
		list_name:function(val){
			if(val == "software"){
	    		this.$refs.upgrade_task_table.refresh();
	    		this.$refs.upgrade_result_table.refresh();
	    		this.taskUrlRb = "";
	    		this.resultUrlRb = "";
	    	}else{
	    		this.taskUrlRb = "${ctx}/task/upgrade/getUpgradeTaskList.action";
	    		this.resultUrlRb = "${ctx}/task/upgrade/getUpgradeDeviceList/rollback.action";
	    	}
		},
		dateValue(newVal){
			var vm = this;
			if(!newVal){
				newVal = [];
				vm.query_task_params.startTime = '';
    			vm.query_task_params.endTime = '';
			}
		},
		dateValueRb(newVal){
			var vm = this;
			if(!newVal){
				newVal = [];
				vm.query_task_params_rb.startTime = '';
    			vm.query_task_params_rb.endTime = '';
			}
		}
	},
	mounted(){
		this.init();
	}
})
</script>