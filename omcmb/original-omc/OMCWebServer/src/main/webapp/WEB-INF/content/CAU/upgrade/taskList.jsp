<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
.deviceItem{
	display:inline-block;
	margin-right:60px;
}
.device_item{
	position:relative;
}
.el-icon-goback:before,.el-icon-status-upgrading:before{
	color:#fff;
	font-size:14px;
}
.advanceQuery{
	margin-left:10px;
}
.device_item .el-radio-button{
	margin-right:10px;
}
.device_item .el-radio-button__inner{
	border:1px solid #F6F7FB;
	background:none;
	padding:5px  15px;
}
.device_item .el-radio-button--mini .el-radio-button__inner,.device_item.el-radio-button:first-child .el-radio-button__inner,.device_item .el-radio-button:last-child .el-radio-button__inner{
	border-radius:13px;
	border-left:1px solid #F6F7FB;
}
.device_item .el-radio-button__orig-radio:checked+.el-radio-button__inner{
	color:#666;
	border-color:#4D84FF;
	background-color:#fff;
	-webkit-box-shadow:none;
}
.list_item .el-tabs__item{
	font-size:14px;
}
.list_item .el-tabs--top{
	border:none;
}
.list_item .el-tabs__nav-scroll{
	margin-left:10px;
}
.list_item{
	position:relative;
}
.list_item .el-radio-button:first-child .el-radio-button__inner{
	border-radius:2px 0 0 2px;
}
.list_item .el-radio-button:last-child .el-radio-button__inner{
	border-radius:0 2px 2px 0;
}
.list_item .el-radio-button .el-icon:before{
	color:#606266;
}
.list_item .el-radio-button__orig-radio:checked+.el-radio-button__inner .el-icon:before{
	color:#fff
}
.list_item .el-radio-button--mini .el-radio-button__inner{
	padding:6px 15px;
}
.list_type{
	position:absolute;
	left:10px;
	top:10px;
}
#cauUpgradePanel .advanceQuery{
	height:28px;
}
#cauUpgradePanel .el-icon-common-query-down,#cauUpgradePanel .arrow-text,#cauUpgradePanel .el-icon-common-query-up{
	line-height:30px;
}
#cauUpgradePanel .pairgrid-query .el-input__inner{
	height:26px;
	line-height:26px;
}
.task_count{
	position:absolute;
	right:30px;
	top:10px;
}
.task_count p,.device_count p{
	display:inline-block;
	margin-left:10px;
}
.task_count p span,.suc_count span,.fail_count span{
	height:28px;
	line-height:30px;
	padding: 0 10px;
	display:inline-block;
	font-size:12px;
	vertical-align:bottom;
}
.task_count p span:nth-child(odd){
	border-radius:4px 0px 0px 4px;
	border-right:0px;
}
.task_count p span:nth-child(even){
	border-radius:0px 4px 4px 0px;
	color:#333;
	font-weight:normal;
}
.task_count i,.suc_count i,.fail_count i{
	margin-right:5px;
	font-size:16px;
}
.suc_count,.fail_count{
	display:inline-block;
	margin-left:10px;
}
.suc_count span:first-child{
	border:1px solid #67D972;
	border-radius:4px 0px 0px 4px;
	border-right:0px;
	color:#67D972 !important;
	background:#EEFFF3;
}
.suc_count span:last-child{
	border:1px solid #67D972;
	border-radius:0px 4px 4px 0px;
	color:#333;
	font-weight:normal;
}
.fail_count span:first-child{
	border:1px solid #E88282;
	border-radius:4px 0px 0px 4px;
	border-right:0px;
	color:#E88282 !important;
	background:#FEF2F2;
}
.fail_count span:last-child{
	border:1px solid #E88282;
	border-radius:0px 4px 4px 0px;
	color:#333;
	font-weight:normal;
}
.suc_count .el-icon-circle-success:before{
	color:#67D972
}
.fail_count .el-icon-circle-close:before{
	color:#E88282
}
#cauUpgradePanel .queryGroup{
	height:28px;
}
.file_select_button .el-radio-button{
	margin-left:15px;
}
.file_select_button .el-radio-button__inner{
	border-left:1px solid #dcdfe6;
	border-radius:2px;
}
.file_select_button{
	margin-bottom:10px;
}
.file_select_button .el-radio-button:last-child .el-radio-button__inner,.file_select_button .el-radio-button:first-child .el-radio-button__inner{
	border-radius:2px;
}
.file_select_button .el-radio-button--mini .el-radio-button__inner{
	padding:6px 15px;
}
.file_item .queryGroup{
	margin-left:10px;
}
.queryInfo label{
	display:block;
	margin-bottom:5px;
	line-height:26px;
}
.queryInfo{
	display:inline-block;
}
.el-badge{
	position:relative;
}
.el-badge__content{
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
.el-icon-star-badge:before{
	color:#F3916C;
}
.el-date-editor .el-range__close-icon{
	line-height:20px;
}
</style>
<div class='panelDefault' id="cauUpgradePanel">
	<el-tabs v-model="activeName" style='height:calc(100% - 2px)'>
		<!-- 升级页面 -->
		<el-tab-pane label='<%=rb.getString("ShengJi")%>&<%=rb.getString("HuiTui")%>' name='upgrade' class='upgrade_item' style='overflow:auto'>
			<div class="circleIcon placeholder-bt" :class="upgradeClass" style="right: 90px;top:8px;" placeholder="<%=rb.getString("ShengJi")%>">		
				<span class="el-icon el-icon-circle-upgrade" @click="addUpgradeTask"></span>
			</div>
			<div style='flex:1;overflow:hidden' class='device_item'>
				<el-ctable v-show="product_type != 'CR-B4860/EU' && product_type != 'CR-B4860/RU' " ref="upgrade_cau_table" id="upgrade_cau_table" row-key="small_cell_code" :url="deviceUrl" :height="height" :query-params="query_cell_params" pagination="true" @selection-change="selectCell">
					<el-table-column type="selection" width="45" reserve-selection=true></el-table-column>
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
						<el-form :model='query_cell_form' ref="query_cell_form" label-position="top">
							<el-query @query="query" @advance-query="advanceQuery" @reset='resetQuery' :placeholder="'<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>'"
							 :ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
								<template slot="form">
									<el-form-item class='deviceItem' label='<%=rb.getString("XiaoZhanBianMa")%>' prop='serial_number'>
										<el-input v-model='query_cell_form.serial_number'  size="mini"></el-input>
									</el-form-item>
									<el-form-item class='deviceItem' label='<%=rb.getString("HostName")%>' prop='host_name'>
										<el-input v-model='query_cell_form.host_name' size="mini"></el-input>
									</el-form-item>
									<el-form-item class='deviceItem' label='<%=rb.getString("SheBeiZu")%>' prop='group_id'>
										<el-select v-model="query_cell_form.group_id" size="mini">
											<el-option v-for="item in groupOptions" :key="item.value" :label="item.text" :value="item.value">
											</el-option>
										</el-select>
									</el-form-item>
									<el-form-item class='deviceItem' label='<%=rb.getString("HuiTuiBanBen")%>' prop='rollback_version'>
										<el-select v-model="query_cell_form.rollback_version" size="mini">
											<el-option v-for="item in rbVersionOptions" :key="item.value" :label="item.text" :value="item.value">
											</el-option>
										</el-select>
									</el-form-item>
									<el-form-item class='deviceItem' label='<%=rb.getString("RuanJianBanBen")%>' prop='software_version'>
										<el-select v-model="query_cell_form.software_version" size="mini">
											<el-option v-for="item in versionOptions" :key="item.value" :label="item.text" :value="item.value">
											</el-option>
										</el-select>
									</el-form-item>
									<el-form-item class='deviceItem' label='<%=rb.getString("SheBeiXingHaoMing")%>' prop='module_type'>
										<el-input v-model='query_cell_form.module_type'  size="mini"></el-input>
									</el-form-item>
								</template>
							</el-query>
						</el-form>
						<div style='height:40px;width:100%;background:#F6F7FB;margin-top:8px;line-height:40px;'>
							<span style='font-weight:bold;margin-left:10px;margin-right:15px;'><%=rb.getString("ChangPinXingHao")%>:</span>
							<el-radio-group size="mini" v-model='product_type' @change="changeProduct">
								<el-radio-button v-for="item in buttonGroups" :label="item.value">{{item.name}}</el-radio-button>
							</el-radio-group>
						</div>
					</template>
				</el-ctable>
				<el-ctable v-show="product_type == 'CR-B4860/EU' " ref="upgrade_eu_table" id="upgrade_eu_table" row-key="serial_number" :url="euDeviceUrl" :height="height" :query-params="query_eu_params" pagination="true" @selection-change="selectCell">
					<el-table-column type="selection" width="45" reserve-selection=true></el-table-column>
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
						<el-query type="normal"  @query="queryEU"  placeholder="'<%=rb.getString("BUJiZhanBianMa")%>/<%=rb.getString("EUJiZhanBianMa")%>'"></el-query>
						<div style='height:40px;width:100%;background:#F6F7FB;margin-top:8px;line-height:40px;'>
							<span style='font-weight:bold;margin-left:10px;margin-right:15px;'><%=rb.getString("ChangPinXingHao")%>:</span>
							<el-radio-group size="mini" v-model='product_type' @change="changeProduct">
								<el-radio-button v-for="item in buttonGroups" :label="item.value">{{item.name}}</el-radio-button>
							</el-radio-group>
						</div>
					</template>
				</el-ctable>
				<el-ctable v-show="product_type == 'CR-B4860/RU' " ref="upgrade_ru_table" id="upgrade_ru_table" row-key="serial_number" :url="ruDeviceUrl" :height="height" :query-params="query_ru_params" pagination="true" @selection-change="selectCell">
					<el-table-column type="selection" width="45" reserve-selection=true></el-table-column>
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
						<el-query type="normal"  @query="queryRU"  placeholder="'<%=rb.getString("BUJiZhanBianMa")%>/<%=rb.getString("RUJiZhanBianMa")%>'"></el-query>
						<div style='height:36px;width:100%;background:#F6F7FB;margin-top:8px;line-height:36px;border:1px solid #E9E9E9'>
							<span style='font-weight:bold;margin-left:10px;margin-right:15px;'><%=rb.getString("ChangPinXingHao")%>:</span>
							<el-radio-group size="mini" v-model='product_type' @change="changeProduct">
								<el-radio-button v-for="item in buttonGroups" :label="item.value">{{item.name}}</el-radio-button>
							</el-radio-group>
						</div>
					</template>
				</el-ctable>
			</div>
			<div style='flex:1;margin-top:20px;overflow:auto' class='list_item'>
				<el-tabs v-model="list_name" type='card' style='height:100%'>
					<el-tab-pane label="<%=rb.getString("RuanJianShengJi")%>" name="software" style="position:relative">
						<el-radio-group v-model="list_type" class="list_type" @change="changeListType" size="mini">
							<el-radio-button label="task"><i class="el-icon el-icon-tasklist" style='margin-right:5px;vertical-align:bottom'></i><%=rb.getString("RenWuLieBiao")%></el-radio-button>
							<el-radio-button label="device"><i class="el-icon el-icon-devicelist" style='margin-right:5px;vertical-align:bottom'></i><%=rb.getString("BackupRestoreSheBeiLieBiao")%></el-radio-button>
						</el-radio-group>
						<el-ctable v-show="showTask" id="cau_upgrade_task_table" ref="cau_upgrade_task_table" time=6 :url="taskUrl" :height="height" :query-params="query_task_params" pagination="true" @load-success="loadSuccessTask">
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
								<el-query style='margin-left:240px;' @query="queryTask" @advance-query="advanceQueryTask" @reset='resetQueryTask' :placeholder="'<%=rb.getString("RenWuMingCheng")%>'"
								:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
									<template slot="form">
										<div class='queryInfo'>
											<label><%=rb.getString("RenWuMingCheng")%></label>
											<el-input v-model="query_task_form.taskName" size="mini" style='width:200px'></el-input>
										</div>
										<div class='queryInfo' style='margin-left:50px;'>
											<label><%=rb.getString("KaiShiShiJian")%></label>
											<el-date-picker v-model='dateValue'  value-format="yyyy-MM-dd HH:mm:ss" type="datetimerange" range-separator="——"  start-placeholder='<%=rb.getString("KaiShiShiJian")%>' end-placeholder='<%=rb.getString("JieShuShiJian")%>'></el-date-picker>
										</div>
									</template>
								</el-query>
								<div class='task_count'>
									<p><span><i class='el-icon el-icon-status-waiting1'></i><%=rb.getString("DengDai")%></span><span>{{waitNum}}</span></p>
									<p><span><i class='el-icon el-icon-status-inProgress'></i><%=rb.getString("JinXingZhong")%></span><span>{{progressNum}}</span></p>
									<p><span><i class='el-icon el-icon-status-suspend'></i><%=rb.getString("ZanTing")%></span><span>{{suspendNum}}</span></p>
									<p><span><i class='el-icon el-icon-status-terminate'></i><%=rb.getString("YiJieShu")%></span><span>{{endNum}}</span></p>
								</div>
							</template>
						</el-ctable>
						<el-ctable v-show="!showTask" ref="cau_result_table" id="cau_result_table" time=6  :url="resultUrl" :height="height" :query-params="query_result_params" pagination="true" @load-success="loadsuccessResult"
						@selection-change="selectDevice">
							<el-table-column type="selection" width="45" :selectable="choseDevice"></el-table-column>
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
								<div class='queryGroup' style='margin-left:250px;'>
									<el-input v-model='query_result_form.searchText' @keyup.enter.native="queryResult" class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%> / <%=rb.getString("HostName")%> / <%=rb.getString("RenWuMingCheng")%>'></el-input>
									<i @click='queryResult' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
								</div>
								<div style='float:right;margin-right:70px;'>
									<p class='suc_count'><span><i class='el-icon el-icon-circle-success'></i><%=rb.getString("ChengGong")%></span><span>{{sucNum}}</span></p>
									<p class='fail_count'><span><i class='el-icon el-icon-circle-close'></i><%=rb.getString("ShiBai")%></span><span>{{failNum}}</span></p>
								</div>
								<div class="circleIcon placeholder-bt" style="top:10px;right:20px;" placeholder="<%=rb.getString("DaoChu")%>" @click="exportResult">		
									<span class="el-icon el-icon-circle-export"></span>
								</div>
							</template>
						</el-ctable>
					</el-tab-pane>
					<el-cmenu ref="menu_task" :data="menus_task" @click="clickMenuTask"></el-cmenu>
					<el-cmenu ref="menu_device" :data="menus_device" @click="clickMenuDevice"></el-cmenu>
				</el-tabs>
			</div>
			<el-bulk ref="deviceBulk" target="cau_result_table" :list="selectionDevice" row-key="serialNumber" show-prop="serialNumber"
						:message="{title:'<%=rb.getString("YiXuanSheBei")%>',subTitle:'<%=rb.getString("XiaoZhanBianMa")%>',clear:'<%=rb.getString("QingKong")%>',cancel:'<%=rb.getString("QuXiao")%>'}">
				<template slot="button">
					<a class="linkbutton linkbutton_trend" @click="restartTask('batch')"><span><%=rb.getString("ChongXinZhiXing")%></span></a>
				</template>
			</el-bulk>
		</el-tab-pane>
		<!-- 文件页面 -->
		<el-tab-pane label='<%=rb.getString("WenJian")%>' name='file' class='file_item' style='position:relative'>
			<el-ctable ref="file_table"  :url="file_url" :height="height" :query-params="query_file_params" pagination="true">
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
				<el-table-column label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>' width="250" prop="product"  v-if="showProductFile"></el-table-column>
				<el-table-column label='<%=rb.getString("WenJianDaXiao")%>' width="200" prop="size"></el-table-column>
				<el-table-column v-if="isCloudCore=='true'?true:false" label='<%=rb.getString("BanBenLeiXing")%>' width="200" prop="toWho" :formatter="towhoFmt"></el-table-column>
				<el-table-column label='<%=rb.getString("ShangChuanShiJian")%>' prop="upload_time"></el-table-column>
				<template slot="toolbar">
					<div class='queryGroup'>
						<el-input v-model='query_file_form.searchText' @keyup.enter.native="queryFile" class='pairgrid-query' placeholder='<%=rb.getString("BanBen")%>'></el-input>
						<i @click='queryFile' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
					</div>
					<div class="circleIcon placeholder-bt" style="right: 30px;top:8px;" placeholder="<%=rb.getString("DaoRuWenJian")%>">		
						<span class="el-icon el-icon-circle-import" @click="importFile"></span>
					</div>
				</template>
			</el-ctable>
		</el-tab-pane>
		<el-cmenu ref="menu_file" :data="menus_file" @click="clickMenuFile"></el-cmenu>
	</el-tabs>
	<el-slide ref="slide" :url="slideUrl" :title="slideTitle" :footer="slideFooter" :header='slideHeader' :position="slidePosition"
	 :height="slideHeight" :modal='slideModal'  :width="slideWidth" :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'"  @cancel='cancelSlide' @ok='saveSlide'>
	 </el-slide>
	 <el-tslide class='tslide' ref="tslide" :url="slideUrl" :title="slideTitle" :footer="false" :position="slidePosition"
		 			:height="slideHeight" :modal='modal'  :width="slideWidth" @ok='' @cancel='cancalImportFile' @operate='' :ok-text="'<%=rb.getString("QueDing")%>'" :cancel-text="'<%=rb.getString("QuXiao")%>'" >
	</el-tslide>
</div>
<script>
var cauFileVue = new Vue({
	el:'#cauUpgradePanel',
	data(){
		return{
			activeName : 'upgrade',
			deviceUrl:'${ctx}/cau/upgrade/queryCellInfos.action',
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
				productValue:'CXA',
				rollback_version:""
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
			},
			query_ru_params:{
				searchText:'',
			},
			groupOptions:[],
			versionOptions:[],
			rbVersionOptions:[],
			buttonGroups:[
				{value:'CXA',name:'CXA'}
			],
			product_type:'CXA',
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
			slideModal:'',
			url:'',
			taskUrl:'${ctx}/cau/upgrade/getUpgradeTaskList.action',
			taskUrlRb:"",
			dateValue:[],
			query_task_params:{
				timeZone:timeZone,
				taskName:'',
				startTime:'',
				endTime:'',
				searchText:'',
				listType:'upgrade'
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
				listType:"rollback"
			},
			query_task_form_rb:{
				taskName:'',
				startTime:'',
				endTime:'',
				searchText:''
			},
			dateValueRb:[],
			file_url:'${ctx}/cell/version/queryfileInfosList.action',
			query_file_params:{
				timeZone:timeZone,
				searchText:'',
				file_type : 'cau',
				productValue : 'CXA'
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
			resultUrl:"${ctx}/cau/upgrade/getUpgradeDeviceList/upgrade.action",
			query_result_params:{
				searchText:"",
				timeZone:timeZone
			},
			query_result_form:{
				searchText:""
			},
			query_result_params_rb:{
				searchText:"",
				timeZone:timeZone
			},
			query_result_form_rb:{
				searchText:""
			},
			resultUrlRb:"",
			showProductFile:true,
			upgradeClass:'',
			selectionDevice:[]
		}
	},
	methods:{
		init(){
			var vm = this;
			if(isJumpToPage){
				if(isJumpToPage.type == 'view'){
					vm.activeName = 'file';
					if(isJumpToPage.file_type == "0") vm.file_type = "upgrade";
					if(isJumpToPage.file_type == "1") vm.file_type = "ca";
					if(isJumpToPage.file_type == "6") vm.file_type = "fpga";
					vm.changeFileType(vm.file_type);
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
				this.upgradeClass = "hidden"
			}else{
				this.upgradeClass = ""
			}
			/* axios.post('${ctx}/task/upgrade/getProductType.action').then(function(response){
				vm.buttonGroups = response.data;
				vm.product_type = vm.buttonGroups[0].value;
				var reg = new RegExp('\\\\',"g");
				if(vm.product_type != 'CR-B4860/EU' && vm.product_type != 'CR-B4860/RU'){
					vm.query_cell_params.productValue = vm.buttonGroups[0].value.includes("CR-B4860") ? vm.buttonGroups[0].value.substr(0,8) : vm.buttonGroups[0].value.replace(reg,'');
				}
				vm.$nextTick(function(){
					vm.deviceUrl = '${ctx}/task/upgrade/queryCellInfos.action?forSelect=1';
					vm.euDeviceUrl = '${ctx}/cell/nxp/queryAllEUInfos.action';
					vm.ruDeviceUrl = '${ctx}/cell/nxp/queryAllRUInfos.action';
				});
			}).catch(function(error){}) */
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : vm.product_type,
				selectType : 'deviceGroup'
			})).then(function(response){
				let data = response.data
				vm.groupOptions = data;
			}).catch(function(error){})
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : vm.product_type,
				selectType : 'version'
			})).then(function(response){
				let data = response.data
				vm.versionOptions = data;
			}).catch(function(error){})
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : vm.product_type,
				selectType : 'rollbackVersion'
			})).then(function(response){
				let data = response.data
				vm.rbVersionOptions = data;
			}).catch(function(error){})
		},
		query(val){
			this.query_cell_params.search_text = val;
		},
		queryEU(val){
			this.query_eu_params.search_text = val;
		},
		queryRU(val){
			this.query_ru_params.search_text = val;
		},
		advanceQuery(){
			this.query_cell_form.search_text = "";
			Object.assign(this.query_cell_params,this.query_cell_form);
		},
		resetQuery(){
			this.$refs.query_cell_form.resetFields();
		},
		setTime(){
			this.rollbackForm.exetime = formatDate(new Date(gloableTime));
			this.$refs.rollbackForm.validateField('exetime');
		},
		addUpgradeTask(){
			var vm = this;
			vm.slideHeader = true
    	    vm.slideTitle = '<%=rb.getString("XinJianRenWu")%>'
    	    vm.slideUrl = "${ctx}/cau/upgrade/toCauAddTaskPage.action"
    	    vm.slideFooter = 'true'
    	    vm.slidePosition = 'top'
    	    vm.slideHeight = '100%'
    	    vm.slideWidth = '100%'
    	    vm.operType = 'addTask'
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
			
			vm.cellData = [];
			if(productIntVal != 'CR-B4860/EU' && productIntVal != 'CR-B4860/RU'){
				if(val.includes("CR-B4860")){
					var index = val.indexOf("/");
					val = val.substring(0,index)
				}
				vm.euDeviceUrl = '';
				vm.ruDeviceUrl = '';
				vm.$refs.upgrade_cau_table.clearSelection();
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
		},
		queryTask(val){
			if(this.list_name == "software"){
				this.query_task_params.searchText = val;
			}else if(this.list_name == "rollback"){
				this.query_task_params_rb.searchText = val;
			}
		},
		advanceQueryTask(){
			var form;
			var params;
			var value;
			if(this.list_name == "software"){
				form = this.query_task_form;
				params = this.query_task_params;
				value = this.dateValue;
			}else if(this.list_name == "rollback"){
				form = this.query_task_form_rb;
				params = this.query_task_params_rb;
				value = this.dateValueRb;
			}
			form.searchText = "";
			if(value != null){
				form.startTime = value[0];
    			form.endTime = value[1];
			}
			Object.assign(params,form);
		},
		resetQueryTask(){
			var form;
			var params;
			var value;
			if(this.list_name == "software"){
				form = this.query_task_form;
				params = this.query_task_params;
				this.dateValue = '';
			}else if(this.list_name == "rollback"){
				form = this.query_task_form_rb;
				params = this.query_task_params_rb;
				this.dateValueRb = '';
			}
			form.taskName = '';
			form.startTime = '';
			form.endTime = '';
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
	    },
	    changeFileType(val){
	    	this.file_url = "";
	    	var urlObj = {
	    			"upgrade" : "${ctx}/cell/version/queryfileInfosList.action?file_type=0",
	    			"ca" : "${ctx}/cell/version/queryfileInfosList.action?file_type=1",
	    			"fpga" : "${ctx}/cell/version/queryfileInfosList.action?file_type=6",
	    			"ap" : "${ctx}/cell/version/queryfileInfosList.action?file_type=11"
	    	}
	    	this.query_file_form.searchText = "";
	    	this.query_file_params.searchText = "";
	    	this.$nextTick(function(){
	    		this.file_url = urlObj[val];
	    	})
	    	if(val == "ap"){
	    		this.showProductFile = false;
	    	}else{
	    		this.showProductFile = true;
	    	}
	    },
	    importFile(){
	    	var vm = this;
    	    vm.slideTitle = '<%=rb.getString("WenJianDaoRu")%>'
    	   /*  vm.slideUrl = "${ctx}/cell/version/toEnodeBImportFilePage.action" */
    	    vm.slideUrl = "${ctx}/cau/upgrade/toCauImportFilePage.action"
    	    vm.slideFooter = true
    	    vm.slideHeader = true
    	    vm.slidePosition = 'top'
    	    vm.slideHeight = '100%'
    	    vm.slideWidth = '100%'
    	    vm.operType = "importFile"
    	    vm.$refs.slide.showSlide(function(){
    	    	vm.modal = false
    	    });
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
	    	axios.post('${ctx}/cau/upgrade/activeTask.action',stringify({
	    		taskId : task_id,
	    		type : type
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			if(vm.list_name == "software"){
	    				vm.$refs.cau_upgrade_task_table.refresh()
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
	    	axios.post('${ctx}/cau/upgrade/suspendTask.action',stringify({
	    		taskId : task_id,
	    		type : type
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			if(vm.list_name == "software"){
	    				vm.$refs.cau_upgrade_task_table.refresh()
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
	    	axios.post('${ctx}/cau/upgrade/terminateUpgradeTask.action',stringify({
	    		taskId : task_id,
	    		type : type
	    	})).then(function(response){
	    		var data = response.data;
	    		if(data["success"]){
	    			if(vm.list_name == "software"){
	    				vm.$refs.cau_upgrade_task_table.refresh()
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
            vm.operType = 'viewTask'
            vm.slideFooter = false;
            if(vm.list_name == "software"){
	    		vm.slideUrl = "${ctx}/cau/upgrade/toCauAddTaskPage.action"
	    	}else{
	    		vm.slideUrl = "${ctx}/task/upgradeRollback/goAddTask.action"
	    	}
	    	vm.slideHeader = true
    	    vm.slidePosition = 'top'
    	    vm.slideHeight = '100%'
    	    vm.slideWidth = '100%'
    	    vm.$refs.slide.showSlide(function(){
    	    	vm.modal = false
    	    });
        },
	    editTask(task_id,type){ //  修改任务
	    	var vm = this;
	    	vm.slideTitle = '<%=rb.getString("XiuGai")%>';
            vm.operType = 'modifyTask'
            vm.slideFooter = true
	    	if(vm.list_name == "software"){
	    		vm.slideUrl = "${ctx}/cau/upgrade/toCauAddTaskPage.action"
	    	}else{
	    		vm.slideUrl = "${ctx}/task/upgradeRollback/goAddTask.action"
	    	}
	    	vm.slideHeader = true
    	    vm.slidePosition = 'top'
    	    vm.slideHeight = '100%'
    	    vm.slideWidth = '100%'
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
	    		axios.post('${ctx}/cau/upgrade/delUpgradeTask.action',stringify({
		    		taskId:task_id,
		    		type:type
		    	})).then(function(response){
		    		var data = response.data;
		    		if(data["success"]){
		    			if(vm.list_name == "software"){
		    				vm.$refs.cau_upgrade_task_table.refresh()
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
			axios.post("${ctx}/cau/upgrade/reTryUpgrade.action",stringify(params)).then(function(response){
				var data = response.data;
				if(data) {
					if(data["success"]){
						vm.$message({
							message: '<%=rb.getString("ChengGong")%>',
							type:'success'
						});
						vm.$refs.cau_upgrade_task_table.refresh()
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
    			'fpga':'CODE_ENB_UPGRADE_FILE hidden'
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
				{label:"<%=rb.getString("XiaZai")%>",cls:"el-icon el-icon-operation-download",code:"download"},
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
				codes[ev.code]()
			}
		},
		handerClose(){
			this.$refs.menu_task.hide();
			this.$refs.menu_file.hide();
		},
		viewFile(){
			var vm = this;
			vm.slideUrl = '${ctx}/cell/version/toEnodeBUpgradeFileEditPage.action',
			vm.slideTitle = '<%=rb.getString("WenJianXinXi")%>';
			vm.slideFooter = false;
			vm.slideHeader = true;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
			vm.operType = 'viewFile';
			vm.$refs.slide.showSlide(function(){
    	    	vm.slideModal = false
    	    });
		},
		editFile(){
			var vm = this;
			vm.slideUrl = '${ctx}/cell/version/toCauUpgradeFileEditPage.action', 
			vm.slideTitle = '<%=rb.getString("WenJianXiuGai")%>';
			vm.slideFooter = true;
			vm.slideHeader = true;
			vm.slidePosition = 'top';
			vm.slideHeight = '100%';
			vm.slideWidth = '100%';
			vm.operType = 'editFile';
			vm.$refs.slide.showSlide(function(){
    	    	vm.slideModal = false
    	    });
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
			vm.slideTitle = '<%=rb.getString("ShengJiHuiTuiRenWu")%>'
    	    vm.slideUrl = "${ctx}/task/upgradeRollback/goAddTask.action"
    	    vm.slideFooter = true
    	    vm.slideHeader = true
    	    vm.slidePosition = 'top'
    	    vm.slideHeight = '100%'
    	    vm.slideWidth = '100%'
    	    vm.operType = 'addTask'
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
	    queryResult(){
	    	if(this.list_name == "software"){
	    		Object.assign(this.query_result_params,this.query_result_form)
	    	}else{
	    		Object.assign(this.query_result_params_rb,this.query_result_form_rb)
	    	}
	    	
	    },
	    clickList(tab){
	    	if(tab.name == "software"){
	    		this.$refs.cau_upgrade_task_table.refresh();
	    		this.$refs.cau_result_table.refresh();
	    		this.taskUrlRb = "";
	    		this.resultUrlRb = "";
	    	}else{
	    		this.taskUrlRb = "${ctx}/cau/upgrade/getUpgradeTaskList.action";
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
	    		params.searchText = vm.query_result_form.searchText;
	    	}else if(vm.list_name == "rollback"){
	    		url = "${ctx}/task/upgrade/exportUpgradeDeviceListToCSV/rollback.action";
	    		params.timeZone = timeZone;
	    		params.searchText = vm.query_result_form_rb.searchText;
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
	    	}else{
	    		vm.selectionDevice.map(function(item){
	    			result.push({[item.taskId]:item.smallCellCode})
	    		})
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
						vm.$refs.cau_upgrade_task_table.refresh()
						vm.$refs.cau_result_table.refresh()
					}else{
						vm.$message.error(data["message"])
					}
				}
			})
	    },
	    selectDevice(selection){
	    	this.selectionDevice = selection;
	    },
	    choseDevice(row,index){
	    	if(row.result == '3'){return true}
	    	return false;
	    }
	},
	watch:{
		list_name:function(val){
			if(val == "software"){
	    		this.$refs.cau_upgrade_task_table.refresh();
	    		this.$refs.cau_result_table.refresh();
	    		this.taskUrlRb = "";
	    		this.resultUrlRb = "";
	    	}else{
	    		this.taskUrlRb = "${ctx}/task/upgrade/getUpgradeTaskList.action";
	    		this.resultUrlRb = "${ctx}/task/upgrade/getUpgradeDeviceList/rollback.action";
	    	}
		}
	},
	mounted(){
		this.init();
	}
})
</script>