<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
#addUpgradeTask .el-form-item__label{
	line-height:26px;
	text-align:left;
}
#addUpgradeTask .modeItem{
	margin-top:20px;
}
#addUpgradeTask .el-select .el-input.is-disabled .el-input__inner{
	min-height:26px;
	max-height:26px;
}
#addUpgradeTask .pairgrid-left,#addUpgradeTask .pairgrid-left .el-ctable{
	border:none;
	border-left:none;
}
.deviceSelectItem .el-radio__label{
	font-size:12px;
}
#addUpgradeTask .el-ctable-toolbar{
	position: relative;
}
#addUpgradeTask .editButton{
	position: absolute;
	right: 130px;
	top: 10px;
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
#addUpgradeTask .editButton i{
	font-size:14px !important;
}
#addUpgradeTask .editButton span{
	font-size:12px;
}
.addDeviceDialog .el-icon-circle-info:before{
	color:#CFCFCF;
}
#addUpgradeTask .el-pairgrid-title{
	top:10px;
}
#addUpgradeTask .el-form-item,.addDeviceDialog .el-form-item{
	display: flex;
	align-items: center;
}
#addUpgradeTask .el-form-item__content,.addDeviceDialog .el-form-item__content{
	margin-left: unset!important;
	width: 100%;
}
#addUpgradeTask .deviceTableBoxCls{
	display:flex;
	height:372px;
	width:100%;
}
#addUpgradeTask .deviceTableBoxCls .transition-box .el-form-item{
	display: inline-block;
	margin-right: 30px;
}
</style>
<!--enb新建升级任务  -->
<div id="addUpgradeTask">
	<el-form :model='ruleForm' :rules="rules" style="padding-top:20px;" ref="ruleForm" :hide-required-asterisk=true label-width="165px">
		<div class="group-title not-extend" style='margin-left:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item class="nameItem" label='<%=rb.getString("RenWuMingCheng")%>' prop='taskname' style='margin-left:45px;margin-top:18px;'>
			<el-input :disabled="showName" maxlength=100 v-model="ruleForm.taskname" size="mini" style="width:348px;height:28px;line-height:28px;"></el-input>
		</el-form-item>
		<div style='width:99%;height:1px;background:#E9E9E9;margin-bottom:30px;'></div>
		<div class="group-title not-extend" style='margin-left:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
		</div>
		<div style='margin-left:45px;width:93%;'>
			<el-form-item style='margin-top:18px;margin-bottom:20px' label="<%=rb.getString("ChangPinXingHao")%>" prop="product" v-show="showProduct">
				<el-select v-model="ruleForm.product" @change="productChange" :disabled="showName">
					<el-option v-for="item in productOptions" :label="item.name" :value="item.value" :key="item.value"></el-option>
				</el-select>
				<el-checkbox :disabled="showName" v-if="ruleForm.product == 'CR-B4860/RU' || ruleForm.product == 'CR-B4860/EU'" v-model="ruleForm.nxpUpgradeFlag" true-label="1" false-label="0" style="margin-left:10px;"><%=rb.getString("QiangZhiShengJi")%></el-checkbox>
			</el-form-item>
			<el-form-item v-if="ruleForm.product != ''" style='margin-top:18px;margin-bottom:20px' label="<%=rb.getString("ShengJiLeiXing")%>" prop="taskType">
				<el-radio-group v-model="ruleForm.taskType" style='margin-top:6px;' :disabled="showName">
					<el-radio border label="1" class="CODE_ENB_UPGRADE_IMAGE hidden"><%=rb.getString("RuanJianShengJi")%></el-radio>
					<el-radio border label="4" v-if="!onlyHasIMG" class="CODE_ENB_UPGRADE_PATCH hidden"><%=rb.getString("CAZhengShuShengJi")%></el-radio>
					<el-radio border label="6" v-if="!onlyHasIMG" class="CODE_ENB_UPGRADE_FPGA hidden"><%=rb.getString("FPGAShengJi")%></el-radio>
					<el-radio border label="7" v-if="isTurbo"><%=rb.getString("APShengJi")%></el-radio>
				</el-radio-group>
			</el-form-item>
			<el-form-item style='margin-bottom:20px' label="<%=rb.getString("ZaiXianZhuangTai")%>" prop="connection_status" v-show='!showName && false'>
				<el-select v-model="ruleForm.connection_status" @change="connectionStatusChange" >
					<el-option v-for="item in onlineOptions" :key="item.value" :label="item.label" :value="item.value"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label='<%=rb.getString("SheBeiZhiDing")%>' style="margin-bottom: 10px;">
				<el-radio-group v-model="deviceType" :disabled="showName" style="padding-top: 5px;">
					<el-radio border label="all"><%=rb.getString("QuanBu")%></el-radio>
					<el-radio border label="select"><%=rb.getString("ZhiDingZhiXing")%></el-radio>
				</el-radio-group>
			</el-form-item>
			<div class="deviceTableBoxCls">
				<div style='border:1px solid #EFF0F2;flex:1;overflow:auto'>
					<el-pairgrid  
						v-show="showDeviceType && !['CR-B4860/RU','CR-B4860/EU'].includes(ruleForm.product)" 
						:id="'select_device_list'" 
						:rownumber="true" 
						ref="add_task_table" 
						:right-url="rightUrl" 
						:left-url="leftUrl" 
						:height="height" 
						row-key="small_cell_code" 
						:query-params="query_cell_params" 
						:title="deviceTitle" 
						:messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}" 
						@selection-change='selectChange' 
						@right-load-success="rightLoadSuccess"
						:readonly="showName"
					>
						<template slot="left">
							<el-table-column type='selection' width="45"></el-table-column>
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
							<el-table-column prop='cell_name' label='<%=rb.getString("HostName")%>' width="220"></el-table-column>
							<el-table-column prop='cell_ip' label='IP' width="180"></el-table-column>
							<el-table-column prop='rollback_version' label='<%=rb.getString("HuiTuiBanBen")%>' width="200"></el-table-column>
							<el-table-column prop='software_version' label='<%=rb.getString("BanBen")%>' width="200"></el-table-column>
							<el-table-column prop='module_type' label='<%=rb.getString("SheBeiXingHaoMing")%>' width="150"></el-table-column>
							<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>'></el-table-column>
						</template>
						<template slot='toolbar'>
							<div style="margin:0px 0px 4px 20px;color:#363B4E"><%=rb.getString("SheBeiLieBiao")%></div>
							<el-form :model='query_cell_form' ref="query_cell_form" label-position="top">
								<div style="display: flex;align-items: center;">
									<el-query type="normal" @query="query" @advance-query="advanceQuery" @reset='resetQuery' :placeholder="'<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>/IP/<%=rb.getString("SheBeiXingHao")%>'"
										:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
										<template v-if="false" slot="form">
											<el-form-item class='deviceItem' label='<%=rb.getString("XiaoZhanBianMa")%>' prop='serial_number'>
												<el-input v-model='query_cell_form.serial_number'  size="mini"></el-input>
											</el-form-item>
											<el-form-item class='deviceItem' label='<%=rb.getString("HostName")%>' prop='host_name'>
												<el-input v-model='query_cell_form.host_name' size="mini"></el-input>
											</el-form-item>
											<el-form-item class='deviceItem' label='<%=rb.getString("SheBeiZu")%>' prop='group_id'>
												<el-select v-model="query_cell_form.group_id" size="mini" multiple collapse-tags>
													<el-option v-for="item in groupOptions" :key="item.value" :label="item.text" :value="item.value">
													</el-option>
												</el-select>
											</el-form-item>
											<el-form-item class='deviceItem' label='<%=rb.getString("HuiTuiBanBen")%>' prop='rollback_version'>
												<el-select v-model="query_cell_form.rollback_version" size="mini" multiple collapse-tags>
													<el-option v-for="item in rbVersionOptions" :key="item.value" :label="item.text" :value="item.value">
													</el-option>
												</el-select>
											</el-form-item>
											<el-form-item class='deviceItem' label='<%=rb.getString("BanBen")%>' prop='software_version'>
												<el-select v-model="query_cell_form.software_version" size="mini" multiple collapse-tags>
													<el-option v-for="item in versionOptions" :key="item.value" :label="item.text" :value="item.value">
													</el-option>
												</el-select>
											</el-form-item>
											<el-form-item class='deviceItem' label='<%=rb.getString("SheBeiXingHao")%>' prop='module_type'>
												<el-input v-model='query_cell_form.module_type'  size="mini"></el-input>
											</el-form-item>
										</template>
									</el-query>
									
									<el-popfilter style="margin: 0 5px;"
										label="<%=rb.getString("SheBeiZu")%>"
										v-model="query_cell_form.group_id"
										:list="groupOptions.filter(item=>item.value !== '').map(item=>{return {label:item.text,value:item.value}})"
										@check-change="advanceQuery">
									</el-popfilter>
									<el-popfilter style="margin: 0 5px;"
										v-show='!showName'
										type="single"
										label="<%=rb.getString("ZaiXianZhuangTai")%>"
										v-model="query_cell_form.connection_status"
										:list="onlineOptions"
										@check-change="advanceQuery">
									</el-popfilter>
									<el-popfilter style="margin: 0 5px;"
										label="<%=rb.getString("HuiTuiBanBen")%>"
										v-model="query_cell_form.rollback_version"
										:list="rbVersionOptions.filter(item=>item.value !== '').map(item=>{return {label:item.text,value:item.value}})"
										@check-change="advanceQuery">
									</el-popfilter>
									<el-popfilter style="margin: 0 5px;"
										label="<%=rb.getString("BanBen")%>"
										v-model="query_cell_form.software_version"
										:list="versionOptions.filter(item=>item.value !== '').map(item=>{return {label:item.text,value:item.value}})"
										@check-change="advanceQuery">
									</el-popfilter>

									<div class="pop-filter-clear" 
										@click="resetQuery">
										<%=rb.getString("QingKongShaiXuan")%>
									</div>
								</div>

								<div class="editButton" size="mini" @click="addBatchSn">
									<i class="el-icon el-icon-batchInput" style="font-size: 14px;padding-right: 5px;"></i>
									<span><%=rb.getString("PiLiangShuRu")%></span>
								</div>
							</el-form>
						</template>
						<template slot='right'>
							<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>'></el-table-column>
							<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>'></el-table-column>
						</template>
					</el-pairgrid>
					<el-pairgrid 
						v-show="showDeviceType && ruleForm.product == 'CR-B4860/EU'" 
						:id="'select_eu_list'" 
						:rownumber="true" 
						ref="add_eu_table" 
						:right-url="euRightUrl" 
						:left-url="euLeftUrl" 
						:height="height" 
						row-key="serial_number" 
						:query-params="query_eu_params" 
						:title="deviceTitle" 
						:messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}" 
						@selection-change='selectChange' 
						@right-load-success="rightLoadSuccess"
						:readonly="showName"
					>
						<template slot="left">
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
						</template>
						<template slot='toolbar'>
							<div style="margin:0px 0px 4px 20px;color:#363B4E"><%=rb.getString("SheBeiLieBiao")%></div>
							
							<div style="display: flex;align-items: center;">
								<el-query type="normal"  @query="queryEU"  placeholder="<%=rb.getString("BUJiZhanBianMa")%> / <%=rb.getString("EUJiZhanBianMa")%>"></el-query>
								
								<el-popfilter style="margin: 0 5px;"
									v-show='!showName'
									type="single"
									label="<%=rb.getString("ZaiXianZhuangTai")%>"
									v-model="query_eu_params.connection_status"
									:list="onlineOptions">
								</el-popfilter>
								<div class="pop-filter-clear" 
									@click="query_eu_params.connection_status = ''">
									<%=rb.getString("QingKongShaiXuan")%>
								</div>
							</div>

							<div class="editButton" size="mini" @click="addBatchSn">
								<i class="el-icon el-icon-plus" style="font-size: 14px;padding-right: 5px;"></i>
								<span><%=rb.getString("PiLiangShuRu")%></span>
							</div>
						</template>
						<template slot='right'>
							<el-table-column prop='bu_serial_number' label='<%=rb.getString("BUJiZhanBianMa")%>'></el-table-column>
							<el-table-column prop='serial_number' label='<%=rb.getString("EUJiZhanBianMa")%>'></el-table-column>
						</template>
					</el-pairgrid>
					<el-pairgrid 
						v-show="showDeviceType && ruleForm.product == 'CR-B4860/RU'" 
						:id="'select_ru_list'" :rownumber="true" 
						ref="add_ru_table" 
						:right-url="ruRightUrl" 
						:left-url="ruLeftUrl" 
						:height="height" 
						row-key="serial_number" 
						:query-params="query_ru_params" 
						:title="deviceTitle" 
						:messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}" 
						@selection-change='selectChange' 
						@right-load-success="rightLoadSuccess"
						:readonly="showName"
					>
						<template slot="left">
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
							<el-table-column label="<%=rb.getString("ZuiDaFaSheGongLv") %>" prop="max_tx_power"></el-table-column>
							<el-table-column label="<%=rb.getString("FangSheZhuangTai") %>" prop="rf_tx_status">
								<template slot-scope="scope">
									<div v-if="scope.row.rf_tx_status == 'false' || scope.row.rf_tx_status == '0'">
										<span class='el-icon el-icon-status-disable' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("GuanBi") %></span>
									</div>
									<div v-if="scope.row.rf_tx_status == 'true' || scope.row.rf_tx_status == '1'">
										<span class='el-icon el-icon-status-enable' style='font-size:22px;'></span><span style='vertical-align:top;margin-left:5px;'><%=rb.getString("KaiQi") %></span>
									</div>
								</template>
							</el-table-column>
							<el-table-column label="<%=rb.getString("MoKuaiXingHao") %>" prop="model_name"></el-table-column>
							<el-table-column label="<%=rb.getString("RuanJianBanBen") %>" prop="software_version"></el-table-column>
						</template>
						<template slot='toolbar'>
							<div style="margin:0px 0px 4px 20px;color:#363B4E"><%=rb.getString("SheBeiLieBiao")%></div>
							
							<div style="display: flex;align-items: center;">
								<el-query type="normal"  @query="queryRU"  placeholder="<%=rb.getString("BUJiZhanBianMa")%> / <%=rb.getString("RUJiZhanBianMa")%>"></el-query>

								<el-popfilter style="margin: 0 5px;"
									v-show='!showName'
									type="single"
									label="<%=rb.getString("ZaiXianZhuangTai")%>"
									v-model="query_ru_params.connection_status"
									:list="onlineOptions">
								</el-popfilter>
								<div class="pop-filter-clear" 
									@click="query_ru_params.connection_status = ''">
									<%=rb.getString("QingKongShaiXuan")%>
								</div>
							</div>

							<div class="editButton" size="mini" @click="addBatchSn">
								<i class="el-icon el-icon-plus" style="font-size: 14px;padding-right: 5px;"></i>
								<span><%=rb.getString("PiLiangShuRu")%></span>
							</div>
						</template>
						<template slot='right'>
							<el-table-column prop='bu_serial_number' label='<%=rb.getString("BUJiZhanBianMa")%>'></el-table-column>
							<el-table-column prop='serial_number' label='<%=rb.getString("RUJiZhanBianMa")%>'></el-table-column>
						</template>
					</el-pairgrid>
					<el-ctable 
						v-show="!showDeviceType && ruleForm.product != 'CR-B4860/RU' && ruleForm.product != 'CR-B4860/EU' " 
						ref="add_task_table_all" 
						id="add_task_table_all" 
						row-key="small_cell_code" 
						:url="deviceUrl" 
						:height="height" 
						:query-params="query_cell_params" pagination="true"
					>
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
						<el-table-column prop='cell_name' label='<%=rb.getString("HostName")%>' width="220"></el-table-column>
						<el-table-column prop='cell_ip' label='IP' width="180"></el-table-column>
						<el-table-column prop='rollback_version' label='Rollback Version' width="200"></el-table-column>
						<el-table-column prop='software_version' label='<%=rb.getString("RuanJianBanBen")%>' width="200"></el-table-column>
						<el-table-column prop='module_type' label='<%=rb.getString("SheBeiXingHao")%>' width="200"></el-table-column>
						<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>'></el-table-column>
						<template slot='toolbar'>
							<div style="margin:0px 0px 4px 20px;color:#363B4E"><%=rb.getString("SheBeiLieBiao")%></div>
							<el-form :model='query_cell_form' ref="query_cell_form" label-position="top">
								<div style="display: flex;align-items: center;">
									<el-query type="normal" @query="query" @advance-query="advanceQuery" @reset='resetQuery' :placeholder="'<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>/IP/<%=rb.getString("SheBeiXingHao")%>'"
										:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
										<template v-if="false" slot="form">
											<el-form-item class='deviceItem' label='<%=rb.getString("XiaoZhanBianMa")%>' prop='serial_number'>
												<el-input v-model='query_cell_form.serial_number'  size="mini"></el-input>
											</el-form-item>
											<el-form-item class='deviceItem' label='<%=rb.getString("HostName")%>' prop='host_name'>
												<el-input v-model='query_cell_form.host_name' size="mini"></el-input>
											</el-form-item>
											<el-form-item class='deviceItem' label='<%=rb.getString("SheBeiZu")%>' prop='group_id'>
												<el-select v-model="query_cell_form.group_id" size="mini" multiple collapse-tags>
													<el-option v-for="item in groupOptions" :key="item.value" :label="item.text" :value="item.value">
													</el-option>
												</el-select>
											</el-form-item>
											<el-form-item class='deviceItem' label='Rollback Version' prop='rollback_version'>
												<el-select v-model="query_cell_form.rollback_version" size="mini" multiple collapse-tags>
													<el-option v-for="item in rbVersionOptions" :key="item.value" :label="item.text" :value="item.value">
													</el-option>
												</el-select>
											</el-form-item>
											<el-form-item class='deviceItem' label='<%=rb.getString("BanBen")%>' prop='software_version'>
												<el-select v-model="query_cell_form.software_version" size="mini" multiple collapse-tags>
													<el-option v-for="item in versionOptions" :key="item.value" :label="item.text" :value="item.value">
													</el-option>
												</el-select>
											</el-form-item>
											<el-form-item class='deviceItem' label='<%=rb.getString("SheBeiXingHao")%>' prop='module_type'>
												<el-input v-model='query_cell_form.module_type'  size="mini"></el-input>
											</el-form-item>
										</template>
									</el-query>

									<el-popfilter style="margin: 0 5px;"
										label="<%=rb.getString("SheBeiZu")%>"
										v-model="query_cell_form.group_id"
										:list="groupOptions.filter(item=>item.value !== '').map(item=>{return {label:item.text,value:item.value}})"
										@check-change="advanceQuery">
									</el-popfilter>
									<el-popfilter style="margin: 0 5px;"
										v-show='!showName'
										type="single"
										label="<%=rb.getString("ZaiXianZhuangTai")%>"
										v-model="query_cell_form.connection_status"
										:list="onlineOptions"
										@check-change="advanceQuery">
									</el-popfilter>
									<el-popfilter style="margin: 0 5px;"
										label="<%=rb.getString("HuiTuiBanBen")%>"
										v-model="query_cell_form.rollback_version"
										:list="rbVersionOptions.filter(item=>item.value !== '').map(item=>{return {label:item.text,value:item.value}})"
										@check-change="advanceQuery">
									</el-popfilter>
									<el-popfilter style="margin: 0 5px;"
										label="<%=rb.getString("BanBen")%>"
										v-model="query_cell_form.rollback_version"
										:list="versionOptions.filter(item=>item.value !== '').map(item=>{return {label:item.text,value:item.value}})"
										@check-change="advanceQuery">
									</el-popfilter>

									<div class="pop-filter-clear" style="margin: 0 5px;" 
										@click="resetQuery">
										<%=rb.getString("QingKongShaiXuan")%>
									</div>
								</div>
							</el-form>
						</template>
					</el-ctable>
					<el-ctable 
						v-show="!showDeviceType && ruleForm.product == 'CR-B4860/EU'"  
						ref="add_eu_table_all" 
						id="add_eu_table_all" 
						row-key="serial_number" 
						:url="euLeftUrl" 
						:height="height" 
						:query-params="query_eu_params" 
						pagination="true"
					 >
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
							<div style="margin:0px 0px 4px 20px;color:#363B4E"><%=rb.getString("SheBeiLieBiao")%></div>
							
							<div style="display: flex;align-items: center;">
								<el-query type="normal"  @query="queryEU"  placeholder="'<%=rb.getString("BUJiZhanBianMa")%>/<%=rb.getString("EUJiZhanBianMa")%>'"></el-query>
								<el-popfilter style="margin: 0 5px;"
									v-show='!showName'
									type="single"
									label="<%=rb.getString("ZaiXianZhuangTai")%>"
									v-model="query_eu_params.connection_status"
									:list="onlineOptions">
								</el-popfilter>
								<div class="pop-filter-clear" 
									@click="query_eu_params.connection_status = ''">
									<%=rb.getString("QingKongShaiXuan")%>
								</div>
							</div>
						</template>
					</el-ctable>
					<el-ctable 
						v-show="!showDeviceType && ruleForm.product == 'CR-B4860/RU'"   
						ref="add_ru_table_all" 
						id="add_ru_table_all" 
						row-key="serial_number" 
						:url="ruLeftUrl" 
						:height="height" 
						:query-params="query_ru_params" 
						pagination="true"
					 >
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
							<div style="margin:0px 0px 4px 20px;color:#363B4E"><%=rb.getString("SheBeiLieBiao")%></div>
							
							<div style="display: flex;align-items: center;">
								<el-query type="normal"  @query="queryRU"  placeholder="'<%=rb.getString("RUJiZhanBianMa")%>/<%=rb.getString("RUJiZhanBianMa")%>'"></el-query>
								<el-popfilter style="margin: 0 5px;"
									v-show='!showName'
									type="single"
									label="<%=rb.getString("ZaiXianZhuangTai")%>"
									v-model="query_ru_params.connection_status"
									:list="onlineOptions">
								</el-popfilter>
								<div class="pop-filter-clear" 
									@click="query_ru_params.connection_status = ''">
									<%=rb.getString("QingKongShaiXuan")%>
								</div>
							</div>
						</template>
					</el-ctable>
				</div>
			</div>
			
			<el-form-item prop='cellCodes' style='margin-bottom:20px;'>
				<el-input v-model='ruleForm.cellCodes' v-show="false"></el-input>
			</el-form-item>
			<div>
				<el-form-item prop='rawMode' style='margin-bottom:5px;' label="<%=rb.getString("WenJianLieBiao") %>" label-width="80px">
					<el-checkbox :disabled="showName" v-model="ruleForm.rawMode" style='margin-top:3px;' v-show="ruleForm.product != 'CR-B4860/RU' && ruleForm.product != 'CR-B4860/EU'"><%=rb.getString("BaoLiuPeiZhi") %></el-checkbox>
				</el-form-item>
			</div>
			<div style='border:1px solid #EFF0F2;'>
				<el-ctable :id="'select_file_list'" ref="add_file_table" :data="fileData" :default-checked="defaultChecked" @row-click="rowClickUpgrade" :url="fileUrl" :height="height" pagination="true" :query-params="fileParams" @load-success="loadSuccessFile">
					<el-table-column label='<%=rb.getString("XuanZe")%>' width="80">
						<template slot-scope="scope">
		              		<div class='tableDiv el-icon el-icon-status-yes selected-status' style="cursor: pointer;"></div>
		            	</template>
					</el-table-column>
					<el-table-column label='<%=rb.getString("BanBen")%>' width="200"  prop="version">
						<template slot-scope="scope">
							<div v-if="scope.row.recommend == '1'" class="el-badge">
								<span>{{scope.row.version}}</span>
								<span class='el-badge__content el-icon el-icon-star-badge'></span>
							</div>
							<div v-else>{{scope.row.version}}</div>
						</template>
					</el-table-column>
					<el-table-column label='<%=rb.getString("ChanPinLeiXingBiaoZhi")%>' width="200" prop="product" v-if="showProduct"></el-table-column>
					<el-table-column label='<%=rb.getString("WenJianMing")%>' prop="file_name"></el-table-column>
					<el-table-column label='<%=rb.getString("WenJianDaXiao")%>' width="150" prop="size"></el-table-column>
					<el-table-column label='<%=rb.getString("ShangChuanShiJian")%>' width="200" prop="upload_time"></el-table-column>
					<el-table-column label='<%=rb.getString("MiaoShu")%>' prop="desc"></el-table-column>
				</el-ctable>
			</div>
			<el-form-item prop='file'>
				<el-input v-model='ruleForm.file' v-show="false"></el-input>
			</el-form-item>
		</div>
		<div style='width:99%;height:1px;background:#E9E9E9;margin-bottom:30px;'></div>
		<div class="group-title not-extend" style='margin-left:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<el-form-item style='display:inline-block;margin-left:45px;' prop='status' class='modeItem'>
			<el-radio-group v-model="ruleForm.status" :disabled="showName">
				<el-radio border label="active"><%=rb.getString("LiJiZhiXing")%></el-radio>
				<el-radio border label="suspend"><%=rb.getString("GuaQi")%></el-radio>
				<!-- 
                <el-radio border label="online"><%=rb.getString("ShangXianZhiXing")%></el-radio>
				 -->
				<el-radio border label="timing" style='margin-bottom:0px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
			</el-radio-group>
		</el-form-item>
		<el-form-item prop='exetime' style='display:inline-block;vertical-align:bottom;margin-left:15px;margin-bottom:32px;' class='timeItem'>
			<el-date-picker value-format="yyyy-MM-dd HH:mm:ss" v-model='ruleForm.exetime' :disabled="setTimeEnable" type="datetime" @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
		</el-form-item>
		<div style='width:99%;height:1px;background:#E9E9E9;margin-bottom:30px;'></div>
		<!-- 接口还未调测 执行策略 -->
		<div class="group-title not-extend" style='margin-left:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("RenWuShuXing")%></span>
		</div>
		<el-form-item style='display:inline-block;margin:20px 45px 10px;width:50%;' label="<%=rb.getString("LiXianSheBei")%>">
			<el-checkbox v-model="ruleForm.isOnlineExecute" true-label="1" false-label="0" :disabled="showName"></el-checkbox> <%=rb.getString("DengDaiShangXianChongShi")%>
		</el-form-item>
		<!-- 设备并发数
		<div class="group-title not-extend" style='margin-left:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingCeLue")%></span>
		</div>
		<br>
		 -->
		<el-form-item style='display:inline-block;margin:20px 45px;width:50%;' :label-width="maxNumLabelWidth" prop='maxConcurrentNumber' label="<%=rb.getString("SheBeiBingFaShu")%>">
			<el-input-number @change="maxConcurrentNumberChange" v-model='ruleForm.maxConcurrentNumber' :min="5" :max="100" :disabled="showName" style="width:200px;height:28px;line-height:28px;"></el-input-number>
		</el-form-item>
	</el-form>
	<el-dialog class='dialogStyle' :title='dialogTitle' width='630px' :visible.sync='listVisible' :append-to-body="true" :close-on-click-modal="false" @close='closeBatchSn'>
		<el-form ref='addListForm' :rules='addListRules' :model='addListForm' label-position="top">
			<div>
				<label><%=rb.getString("XiaoZhanBianMa")%></label>
				<el-form-item prop='serialNumber' style="margin-bottom:22px;">
					<el-input v-model='addListForm.serialNumber' type='textarea' :rows="4" style='margin-top:5px;'></el-input>
				</el-form-item>
				<p style='display:flex;color:#BBB'><span class='el-icon el-icon-circle-info' style='font-size:14px;'></span><span style="font-size:12px;"><%=rb.getString("eNBZhuCeTiShiWenZi") %></span></p>
			</div>
			<div style='margin-top:45px;'>
				<el-button @click='saveBatchSn' type="primary"><%=rb.getString("QueDing")%></el-button>
				<el-button @click='closeBatchSn'><%=rb.getString("QuXiao")%></el-button>
			</div>
		</el-form>
	</el-dialog>
</div>

<script type="text/javascript">
var addUpgrade = new Vue({
	el:'#addUpgradeTask',
	data(){
		var vm = this;
		var validateName = (rule,value,callback) => {
			if(value === ''){
				callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
			}else if(value.trim() == vm.defaultTaskName){
				callback();
			}else{
				axios.post('${ctx}/task/upgrade/taskNameExist.action',stringify({
					taskName:vm.ruleForm.taskname.trim(),
					taskType:vm.ruleForm.taskType
				})).then(function(response){
					var data = response.data;
					if(data["success"]){
						if(data["message"] == "true"){
							callback(new Error('<%=rb.getString("RenWuMingChengYiCunZai")%>'))
						}else{
							callback();
						}
					}
				}).catch(function(error){
					callback()
				})
			}
		};
		var validateTime = (rule,value,callback) => {
			if(this.ruleForm.status !== 'timing'){
				callback()
			}else{
				if(value === '' || value === null || value === undefined){
					callback(new Error('<%=rb.getString("QingXuanZeShiJian")%>'))
				}else{
					callback();
				}
			}
		}
		var validateCodes = (rule,value,callback) => {
			if(this.deviceType == 'all'){
				callback()
			}else{
				if(value === '' || value === null || value === undefined){
					callback(new Error('<%=rb.getString("QingXuanZeSheBei")%>'))
				}else{
					callback();
				}
			}
		}
		var validatorNum = (rule,value,callback) => {
			var serialNumber = value, temp = /^(\d|[a-zA-Z]|-|\s){1,30}$/,
				list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});
	
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
		};
		return {
			leftUrl:'',
			rightUrl:'',
			euLeftUrl:'',
			euRightUrl:'',
			ruLeftUrl:'',
			ruRightUrl:'',
			deviceTitle:['','<%=rb.getString("YiXuan")%>'],
			height:'370px',
			width:'90%',
			fileUrl:'',
			fileParams:{
				timeZone:timeZone,
				productValue:'',
				isShowSlave:false,
				file_type:''
			},
			setTimeEnable:true,
			rowData : [],
			ruleForm:{
				taskname:'${addTaskName}',
				cellCodes:'',
				file:'',
				status:'active',
				exetime:'',
				rawMode:true,
				taskType:"",
				product:"",
				nxpUpgradeFlag:'0',
				euSerialNumber:'',
				ruSerialNumber:'',
				maxConcurrentNumber:20,
				connection_status: '',
				isOnlineExecute: ''
			},
			rules:{
				taskname:[
					{validator:validateName,trigger:'blur'}
				],
				cellCodes:[
					{validator:validateCodes,trigger:'change'}
				],
				file:[
					{required:true,message:'<%=rb.getString("QingXianXuanZeWenJian")%>',trigger:'change'}
				],
				exetime:[
					{type:'date',validator:validateTime,trigger:'change'}
				]
			},
			defaultChecked:[],
			defaultTaskName:'',
			pickerOptions:{
				disabledDate(time){
					return time.getTime()< Date.now()-8.64e7;
				}
			},
			deviceUrl:'',
			query_eu_params:{
				search_text:'',
				connection_status: ''
			},
			query_ru_params:{
				search_text:'',
				connection_status: ''
			},
			query_cell_params:{
				group_id:'',
				serial_number:'',
				host_name:'',
				software_version:'',
				search_text:'',
				module_type:'',
				productValue:'',
				rollback_version:'',
				connection_status: ''
			},
			query_cell_form:{
				group_id:[],
				serial_number:'',
				host_name:'',
				software_version:[],
				module_type:'',
				rollback_version:[],
				connection_status: ''
			},
			groupOptions:[],
			versionOptions:[],
			rbVersionOptions:[],
			firstFlag:true,
			firstFlagFile:true,
			showName:false,
			productOptions:[],
			deviceType:'select',
			showDeviceType:true,
			selection:[],
			showProduct:true,
			fileData:[],
			addListForm:{
				serialNumber:'',
				type:'input'
			},
			addListRules:{
				serialNumber:[
					{validator:validatorNum,trigger:'change'}
				]
			},
			listVisible:false,
			dialogTitle:"<%=rb.getString("TianJia")%>",
			onlineOptions:[
				{label:"<%=rb.getString("QuanBu")%>",value:""},
				{label:"<%=rb.getString("ZaiXian")%>",value:"1"},
				{label:"<%=rb.getString("LiXian")%>",value:"0"}
			],
		}
	},
	computed: {
		onlyHasIMG() { 
			var vm = this;
			// 非RTS和RTD
			return !['FAP','FAP/\\w+(BS81)\\w+/(DC|SC)'].includes(vm.ruleForm.product);
		},
		isTurbo() {
			var vm = this;
			// Turbo基站 -- 通过全局变量判断
			return isLWAEnable == true;
		},
		maxNumLabelWidth(){
			return isLocalZH == true ? '200px' : '290px';
		},
	},
	methods:{ 
		init:function(){
			var vm = this;
			var reg = new RegExp('\\\\',"g");
			vm.productOptions = enbFileVue.productTypeList;
			var product = isJumpToPage ? isJumpToPage.product : enbFileVue.product_type;
			var productFmt = product.includes("CR-B4860") ? product.substr(0,8) : product.replace(reg,'');
			var fileTypeObj = {
					"CR-B4860/EU": "7",
					"CR-B4860/BU": "10",
					"CR-B4860/RU": "8"
			}
			if(enbFileVue.operType == "modifyTask" ||　enbFileVue.operType == "viewTask"){
				axios.post('${ctx}/task/upgrade/getTask.action',stringify({
					taskId : enbFileVue.rowDataTask.TASK_ID,
					timeZone : timeZone
				})).then(function(response){
					var data = response.data;
					vm.ruleForm.taskname = data.TASK_NAME;
					vm.ruleForm.taskType = data.TASK_TYPE +"";
					vm.ruleForm.file = data.FILE_ID;
					vm.ruleForm.status = data.CREATE_STATUS;
					if(data.CREATE_STATUS == 'timing'){
						vm.ruleForm.exetime = data.CREATE_TIME;
					}
					vm.ruleForm.rawMode = data.IS_KEEP_CONFIG == 'true'?false:true;
					vm.defaultChecked = [data.FILE_ID+''];
					vm.defaultTaskName = data.TASK_NAME;
					vm.deviceType = data.selectAll == "true"?"all":"select";
					vm.ruleForm.product = data.PRODUCT_TYPE;
					product = data.PRODUCT_TYPE;
					productFmt = data.PRODUCT_TYPE.includes("CR-B4860") ? data.PRODUCT_TYPE.substr(0,8) : data.PRODUCT_TYPE.replace(reg,'');
					vm.ruleForm.nxpUpgradeFlag = data.NXP_UPGRADE_FLAG;
					vm.ruleForm.maxConcurrentNumber = data.maxConcurrentNumber;
					vm.ruleForm.isOnlineExecute = data.isOnlineExecute;
					vm.fileParams.productValue = productFmt;
					if(vm.deviceType == 'select' && vm.ruleForm.product != 'CR-B4860/EU' && vm.ruleForm.product != 'CR-B4860/RU'){
						vm.query_cell_params.productValue = productFmt;
					}
					vm.fileParams.file_type = data.PRODUCT_TYPE.includes("CR-B4860") ? fileTypeObj[data.PRODUCT_TYPE] : '';
					vm.$nextTick(function(){
						
						vm.deviceUrl = '${ctx}/task/upgrade/queryCellInfos.action?forSelect=1';
						vm.leftUrl = '${ctx}/task/upgrade/queryCellInfos.action?forSelect=1';
						vm.$refs.ruleForm.clearValidate();
						vm.rightUrl = '${ctx}/task/upgrade/getTaskSelectedList.action?taskId=' + enbFileVue.rowDataTask.TASK_ID;
						vm.euLeftUrl = '${ctx}/cell/nxp/queryAllEUInfos.action';
						vm.euRightUrl = '${ctx}/cell/nxp/querySelectedEUInfos.action?type=modify&taskId=' + enbFileVue.rowDataTask.TASK_ID ;
						vm.ruLeftUrl = '${ctx}/cell/nxp/queryAllRUInfos.action';
						vm.ruRightUrl = '${ctx}/cell/nxp/querySelectedRUInfos.action?type=modify&taskId=' + enbFileVue.rowDataTask.TASK_ID ;
						initForm(vm.$refs.ruleForm);
    		    	});
					if(enbFileVue.operType == "viewTask"){
						vm.showName = true;
						vm.fileUrl = ''
						axios.post('${ctx}/cell/version/queryfileInfosList.action',stringify({
							timeZone:timeZone,
							productValue:vm.fileParams.productValue,
							isShowSlave:false,
							fileId:vm.ruleForm.file,
							file_type:vm.fileParams.file_type,
							page:1,
							rows:10,
							sort:"",
							order:""
						})).then(function(response){
							var data = response.data;
							vm.fileData = data.rows;
							setTimeout(function(){
								vm.$refs.add_file_table.setCurrentRow(vm.fileData[0])
							},500)
						})
					}
				})
			}else{
				var typeObj = {
						"0" : "1",
						"1" : "4",
						"6" : "6",
						"7" : "1", // 10 bu
						"8" : "1", //8 ru
						"10" : "1", //7 eu
				}
				vm.ruleForm.product = product;
				if(vm.ruleForm.product != 'CR-B4860/EU' && vm.ruleForm.product != 'CR-B4860/RU'){
					vm.query_cell_params.productValue = productFmt;
					vm.$nextTick(function(){
						if(!isJumpToPage) vm.$refs.add_task_table.appendCheckedRows(enbFileVue.cellData);
					});
					
				}else{
					if(vm.ruleForm.product == 'CR-B4860/EU'){
						vm.$nextTick(function(){
							if(!isJumpToPage) vm.$refs.add_eu_table.appendCheckedRows(enbFileVue.cellData);
						});
						
					}else{
						vm.$nextTick(function(){
							if(!isJumpToPage) vm.$refs.add_ru_table.appendCheckedRows(enbFileVue.cellData);
						});
					}
				}
			
				vm.fileParams.productValue = productFmt;
				vm.fileParams.file_type = product.includes("CR-B4860") ? fileTypeObj[enbFileVue.product_type] : '';
				vm.$nextTick(function(){
					vm.deviceUrl = '${ctx}/task/upgrade/queryCellInfos.action?forSelect=1';
					vm.leftUrl = '${ctx}/task/upgrade/queryCellInfos.action?forSelect=1';
					vm.euLeftUrl = '${ctx}/cell/nxp/queryAllEUInfos.action';
					vm.ruLeftUrl = '${ctx}/cell/nxp/queryAllRUInfos.action';
					vm.$refs.ruleForm.clearValidate();
					vm.ruleForm.taskType = isJumpToPage ? typeObj[isJumpToPage.file_type] : "1";
					setTimeout(function(){
						initForm(vm.$refs.ruleForm);
					},2000)
				});
			}
			vm.getQueryData(product);
			
		},
		query(val){
			this.advanceQuery();
			this.query_cell_params.search_text = val;
		},
		queryEU(val){
			this.query_eu_params.search_text = val;
		},
		queryRU(val){
			this.query_ru_params.search_text = val;
		},
		advanceQuery(){
			//this.query_cell_form.search_text = "";
			var arrGroup = this.query_cell_form.group_id;
			var resGroup = arrGroup.indexOf("");
			var arrVer = this.query_cell_form.software_version;
			var resVer = arrVer.indexOf("");
			var arrRbVer = this.query_cell_form.rollback_version;
			var resRbVer = arrRbVer.indexOf("");
			
			this.query_cell_params.group_id = resGroup == -1 ? arrGroup.join(",") : '';
			this.query_cell_params.software_version = resVer == -1 ? arrVer.join(",") : '';
			this.query_cell_params.rollback_version = resRbVer == -1 ? arrRbVer.join(",") : '';
			
			this.query_cell_params.serial_number = this.query_cell_form.serial_number;
			this.query_cell_params.host_name = this.query_cell_form.host_name;
			this.query_cell_params.module_type = this.query_cell_form.module_type;
			this.query_cell_params.connection_status = this.query_cell_form.connection_status;
			
			//Object.assign(this.query_cell_params,this.query_cell_form);
		},
		resetQuery(){
			this.$refs.query_cell_form.resetFields();

			Object.assign(this.query_cell_form, {
				group_id: [], 
				connection_status: '', 
				rollback_version: [], 
				software_version: []
			});

			Object.assign(this.query_cell_params, {
				group_id: '', 
				connection_status: '', 
				rollback_version: '', 
				software_version: '',
			});
		},
		rowClickUpgrade(row){
	        this.rowData = row;
	    },
		submit(){
	    	var vm = this;
	    	var message = '<%=rb.getString("ChengGong")%>';
	    	vm.$refs.ruleForm.validate((valid) => {
	    		if(valid){
	    			var params = {};
	    			if(vm.deviceType == 'all'){
	    				params.selectAll = "true"
	    			}else{
	    				params.selectAll = "false"
	    				params.cellCodes = vm.ruleForm.cellCodes;
						params.euSerialNumber = vm.ruleForm.euSerialNumber;
						params.ruSerialNumber = vm.ruleForm.ruSerialNumber;
	    			}
	    			params.timeZone = timeZone;
	    			params.taskName = vm.ruleForm.taskname;
	    			params.status = vm.ruleForm.status;
	    			if(vm.ruleForm.status == 'timing'){
	    				params.time = vm.ruleForm.exetime;
	    			}
					if(vm.ruleForm.product == 'CR-B4860/EU' || vm.ruleForm.product == 'CR-B4860/RU'){
						params.nxpUpgradeFlag = vm.ruleForm.nxpUpgradeFlag;
					}
	    			params.fileId = vm.ruleForm.file;
	    			params.taskType = vm.ruleForm.taskType;
	    			params.rawMode = vm.ruleForm.rawMode == true ? 'false' : 'true';
	    			params.productValue = vm.ruleForm.product;
	    			params.maxConcurrentNumber = vm.ruleForm.maxConcurrentNumber;
	    			params.isOnlineExecute = vm.ruleForm.isOnlineExecute;
	    			if(enbFileVue.operType == "modifyTask"){
	    				url = '${ctx}/task/upgrade/updateTask.action'
	    	    		params.taskId = enbFileVue.rowDataTask.TASK_ID
    	    			if(!isFormChanged(vm.$refs.ruleForm)){
    						vm.$alert('<%=rb.getString("CanShuZhiMeiYouBianHua")%>','<%=rb.getString("TiShi")%>',{
    							confirmButtonText:'<%=rb.getString("QueDing")%>',
    							type:'warning'
    						}).then().catch();
    						return;
       	    			}
	    	    		vm.saveTask(url,params,message);
	    			}else{
	    				url = '${ctx}/task/upgrade/addTask.action'
       	    			vm.$confirm("<%=rb.getString("QueRenXinJianRenWu")%>",'<%=rb.getString("QueRen")%>',{
       						customClass:'warningConfirm',
       						confirmButtonText:'<%=rb.getString("QueDing")%>',
       						cancelButtonText:'<%=rb.getString("QuXiao")%>',
       						type:'warning',
       						closeOnClickModal:false
       					}).then(() => {
       						vm.saveTask(url,params,message);
       					}).catch(() => {})
	    			}
	    		}else{
	    			return false;
	    		}
	    	})
		},
		saveTask(url,params,message){
			var vm = this;
            // 防止多次提交
            if(enbFileVue.slideSubmitLoading)return

            enbFileVue.slideSubmitLoading = true;
			axios.post(url,stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
					vm.$message({
						message:message,
						type:'success',
					})
                    enbFileVue.$refs.slide.hide();
                    enbFileVue.list_name = "software";
                    enbFileVue.$refs.upgrade_cell_table.clearSelection();
                    enbFileVue.$refs.upgrade_task_table.refresh();
                    isJumpToPage = '';
				}else{
					vm.$message.error(data["message"]);
                    enbFileVue.slideSubmitLoading = false;
				}
			})
		},
		cancel(){
			var vm = this;
			var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
			if(isFormChanged(this.$refs.ruleForm)){
				vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
					customClass:'warningConfirm',
					confirmButtonText:'<%=rb.getString("QueDing")%>',
					cancelButtonText:'<%=rb.getString("QuXiao")%>',
					type:'warning',
					closeOnClickModal:false
				}).then(() => {
					enbFileVue.$refs.slide.hide();
					isJumpToPage = '';
				}).catch(() => {})
			}else{
				enbFileVue.$refs.slide.hide();
				isJumpToPage = '';
			}
		},
		setTime(){
			this.ruleForm.exetime = formatDate(new Date(gloableTime));
			this.$refs.ruleForm.validateField('exetime');
		},
		loadSuccessFile(data){
			var vm = this;
			if(enbFileVue.operType == "modifyTask" || enbFileVue.operType == "viewTask"){
				axios.post('${ctx}/task/upgrade/getTask.action',stringify({
					taskId : enbFileVue.rowDataTask.TASK_ID,
					timeZone : timeZone
				})).then(function(response){
					var result = response.data;
					data.rows.map(function(item){
						if(result.FILE_ID == item.id){
							vm.$refs.add_file_table.setCurrentRow(item)
						}
					})
					if(vm.firstFlagFile){
						initForm(vm.$refs.ruleForm);
						vm.firstFlagFile = false;
					}
				})
			}
			if(isJumpToPage){
				data.rows.map(function(item){
					if(isJumpToPage.vid == item.id){
						vm.$refs.add_file_table.setCurrentRow(item);
						vm.ruleForm.file = isJumpToPage.vid;
					}
				})
			}
		},
		selectChange(selection){
			this.selection = selection;
		},
		rightLoadSuccess(){
			var vm = this;
			if(vm.firstFlag){
				initForm(vm.$refs.ruleForm);
				vm.firstFlag = false;
			}
		},
		productChange(val){
			var vm = this,
				reg = new RegExp('\\\\',"g");
			if(vm.showDeviceType){

				if(val != 'CR-B4860/EU' && val != 'CR-B4860/RU' ){
					vm.$nextTick(function(){
						vm.$refs.add_task_table.clear();
					});
				}else{
					if(val == 'CR-B4860/EU'){
						vm.$nextTick(function(){
							vm.$refs.add_eu_table.clear();
						});
					}else{
						vm.$nextTick(function(){
							vm.$refs.add_ru_table.clear();
						})
					}
				}
			}
			vm.getQueryData(val);
		},
		connectionStatusChange(val){
			var vm = this;
			if(vm.ruleForm.product == 'CR-B4860/RU'){
				vm.query_ru_params.connection_status = val;
			}else if(vm.ruleForm.product == 'CR-B4860/EU'){
				vm.query_eu_params.connection_status = val;
			}else{
				vm.query_cell_params.connection_status = val;
			}
		},
		getQueryData(product){
			var vm = this;
			var reg = new RegExp('\\\\',"g");
			var productVal = product.includes("CR-B4860") ? product.substr(0,8) : product.replace(reg,'');
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : productVal,
				selectType : 'deviceGroup'
			})).then(function(response){
				let data = response.data
				vm.groupOptions = data || [];
			}).catch(function(error){})
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : productVal,
				selectType : 'version'
			})).then(function(response){
				let data = response.data
				vm.versionOptions = data || [];
			}).catch(function(error){})
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : productVal,
				selectType : 'rollbackVersion'
			})).then(function(response){
				let data = response.data
				vm.rbVersionOptions = data || [];
			}).catch(function(error){})
		},
		addBatchSn(){
			this.listVisible = true;
		},
		closeBatchSn(){
			this.listVisible = false;
			this.$refs.addListForm.resetFields();
		},
		saveBatchSn(){
			var vm = this, saveBatchUrl = '', snStr = vm.addListForm.serialNumber,
				list = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});

			var params = {
					serialNumbers : list.join(";"),
					productType : vm.ruleForm.product
			}
			
			if(vm.ruleForm.product == 'CR-B4860/RU' || vm.ruleForm.product == 'CR-B4860/EU'){
				saveBatchUrl = '${ctx}/cell/cpeinfos/getEUOrRUTaskCheckSNList.action';
			}else {
				saveBatchUrl = '${ctx}/cell/cpeinfos/getTaskCheckSNList.action';
			}
			vm.$refs.addListForm.validate((valid) => {
				if(valid){
					axios.post(saveBatchUrl, stringify(params)).then((res)=>{
						var data = res.data;
						if(data && data.length > 0){		
							if(vm.ruleForm.product != 'CR-B4860/EU' && vm.ruleForm.product != 'CR-B4860/RU'){
								vm.$refs.add_task_table.appendCheckedRows(data);				
							}else{
								if(vm.ruleForm.product == 'CR-B4860/EU'){
									vm.$refs.add_eu_table.appendCheckedRows(data);
								}else{
									vm.$refs.add_ru_table.appendCheckedRows(data);
								}
							}
							vm.closeBatchSn();
						}else{
							vm.$message('<%=rb.getString("MeiYouKePiPeiSheBei")%>')
						}
					})
				}
			})



			
		},
		// 最大并发数失焦事件
		maxConcurrentNumberChange(event){
			var vm = this,
				value = vm.ruleForm.maxConcurrentNumber;
			if(value == undefined || value == '' || value == null){
				vm.$nextTick(()=>{
					vm.ruleForm.maxConcurrentNumber = 5;
				})
			}
		},
	},
	watch:{
		rowData(newVal){
			this.ruleForm.file = newVal.id
		},
		"ruleForm.status":function(newVal){
			if(enbFileVue.operType =='viewTask' || newVal != 'timing'){
				this.setTimeEnable = true;
			}else{
				this.setTimeEnable = false;
			}
			this.$refs.ruleForm.validateField('exetime')
		},
		"ruleForm.taskType":function(val){
			var vm = this,
				reg = new RegExp('\\\\',"g");
			if(enbFileVue.operType != "viewTask"){
				vm.fileUrl = "";
				var urlObj = {
						"1" : "${ctx}/cell/version/queryfileInfosList.action",
						"4" : "${ctx}/cell/version/queryfileInfosList.action?file_type=1",
						"6" : "${ctx}/cell/version/queryfileInfosList.action?file_type=6",
						"7" : "${ctx}/cell/version/queryfileInfosList.action?file_type=11"
				}
				vm.showProduct = val == "7" ? false : true;
				vm.fileParams.productValue = val == "7" ? "" : this.ruleForm.product.includes("CR-B4860") ?  this.ruleForm.product.substr(0,8) : this.ruleForm.product.replace(reg,'');
				if(val == "7"){
					vm.query_cell_params.productValue = "";
				}else{
					vm.query_cell_params.productValue = vm.ruleForm.product.includes("CR-B4860") ?  vm.ruleForm.product.substr(0,8) : vm.ruleForm.product.replace(reg,'');
				}
				vm.$nextTick(function(){
					vm.fileUrl = urlObj[val];
				})
			}
		},
		deviceType:function(val){
			if(val == 'all'){
				this.showDeviceType = false;
			}else{
				this.showDeviceType = true;
			}
			this.resetQuery();
		},
		"ruleForm.product":function(val){
			var vm = this;
			var fileTypeObj = {
					"CR-B4860/EU" : "7",
					"CR-B4860/BU" : "10",
					"CR-B4860/RU" : "8"
			}
			var reg = new RegExp('\\\\',"g");
			if(vm.ruleForm.taskType == "7"){
				vm.query_cell_params.productValue = val.includes("CR-B4860") ? val.substr(0,8) : val.replace(reg,'');
			}else{
				vm.fileParams.productValue = val.includes("CR-B4860") ? val.substr(0,8) : val.replace(reg,'');
				vm.fileParams.file_type = fileTypeObj[val];
				vm.query_cell_params.productValue = val.includes("CR-B4860") ? val.substr(0,8) : val.replace(reg,'')
			}
			
		},
		selection(){
			var vm = this,
			cellCodeList=[],
			euSerialNumberList=[],
			ruSerialNumberList=[]; 

			if(vm.ruleForm.product != 'CR-B4860/EU' && vm.ruleForm.product != 'CR-B4860/RU'){
				var data = vm.$refs.add_task_table.getData();
				if(data.length != 0){
					data.map(function(item){
						cellCodeList.push(item.small_cell_code);
					})
				}
				vm.ruleForm.cellCodes = cellCodeList.join(',');
			}else{
				if(vm.ruleForm.product == 'CR-B4860/EU'){
					var data = vm.$refs.add_eu_table.getData();
					if(data.length != 0){
						data.map(function(item){
							euSerialNumberList.push(item.serial_number)
							if(!cellCodeList.includes(item.bu_serial_number)){
								cellCodeList.push(item.bu_serial_number);
							}
						})
					}
					vm.ruleForm.cellCodes = cellCodeList.join(',');
					vm.ruleForm.euSerialNumber = euSerialNumberList.join(',');
				}else{
					var data = vm.$refs.add_ru_table.getData();
					if(data.length != 0){
						data.map(function(item){
							ruSerialNumberList.push(item.serial_number)
							if(!cellCodeList.includes(item.bu_serial_number)){
								cellCodeList.push(item.bu_serial_number);
							}
						})
					}
					vm.ruleForm.cellCodes = cellCodeList.join(',');
					vm.ruleForm.ruSerialNumber = ruSerialNumberList.join(',');
				}
			}
		}
	},
	mounted(){
		this.init();
		eventBus.$off('add-task').$on('add-task',this.submit);
		eventBus.$off('cancel-add-task').$on('cancel-add-task',this.cancel);
	}
})
</script>