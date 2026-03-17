<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
#addCauTask .el-form-item__label{
	line-height:26px;
	width:110px;
	text-align:left;
}
#addCauTask .modeItem{
	margin-top:20px;
}
#addCauTask .el-select .el-input.is-disabled .el-input__inner{
	min-height:26px;
	max-height:26px;
}
.fileContent .el-radio{
	margin-right:30px;
}
.nameItem .el-form-item__error{
	margin-left:110px;
}
#addCauTask .pairgrid-left,#addCauTask .pairgrid-left .el-ctable{
	border:none;
	border-left:none;
}
.deviceSelectItem .el-radio__label{
	font-size:12px;
}
</style>

<%-- 新建升级任务 --%>
<div id="addCauTask">
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm" :hide-required-asterisk=true>
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
		<div class='fileContent' style='margin-left:45px;width:93%;'>
			<el-form-item style='margin-top:18px;margin-bottom:20px' label="<%=rb.getString("ChangPinXingHao")%>" prop="product" v-show="showProduct">
				<el-select v-model="ruleForm.product" @change="productChange" :disabled="showName">
					<el-option v-for="item in productOptions" :label="item.name" :value="item.value" :key="item.value"></el-option>
				</el-select>
				<el-checkbox :disabled="showName" v-if="ruleForm.product == 'CR-B4860/RU' || ruleForm.product == 'CR-B4860/EU'" v-model="ruleForm.nxpUpgradeFlag" true-label="1" false-label="0" style="margin-left:10px;"><%=rb.getString("QiangZhiShengJi")%></el-checkbox>
			</el-form-item>
			<p>eNBs</p>
			<div style='display:flex;height:372px;width:100%;'>
				<div style='width:220px;border:1px solid #EFF0F2;' class='deviceSelectItem'>
					<p style='height:30px;background:#F6F7FB;text-align:center;line-height:30px;'><%=rb.getString("BackupRestoreSheBeiZhiDing")%></p>
					<el-radio :disabled="showName" v-model="deviceType" label="all" style='margin:60px 0px 20px 30px;'><%=rb.getString("QuanBu")%></el-radio>
					<el-radio :disabled="showName" v-model="deviceType" label="select"><%=rb.getString("ZhiDingZhiXing")%></el-radio>
				</div>
				<div style='border:1px solid #EFF0F2;border-left:none;flex:1;overflow:auto'>
					<el-pairgrid  
						v-show="showDeviceType && ruleForm.product != 'CR-B4860/RU' && ruleForm.product != 'CR-B4860/EU' " 
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
							<el-table-column type="selection" width="45"></el-table-column>
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
							<el-table-column prop='software_version' label='<%=rb.getString("BanBen")%>' width="200"></el-table-column>
							<el-table-column prop='module_type' label='<%=rb.getString("SheBeiXingHaoMing")%>' width="150"></el-table-column>
							<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>' width="200"></el-table-column>
						</template>
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
										<el-form-item class='deviceItem' label='<%=rb.getString("BanBen")%>' prop='software_version'>
											<el-select v-model="query_cell_form.software_version" size="mini">
												<el-option v-for="item in versionOptions" :key="item.value" :label="item.text" :value="item.value">
												</el-option>
											</el-select>
										</el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("SheBeiXingHao")%>' prop='module_type'>
											<el-input v-model='query_cell_form.module_type'  size="mini"></el-input>
										</el-form-item>
									</template>
								</el-query>
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
						</template>
						<template slot='toolbar'>
							<el-query type="normal"  @query="queryEU"  placeholder="<%=rb.getString("BUJiZhanBianMa")%> / <%=rb.getString("EUJiZhanBianMa")%>"></el-query>
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
							<el-query type="normal"  @query="queryRU"  placeholder="<%=rb.getString("BUJiZhanBianMa")%> / <%=rb.getString("RUJiZhanBianMa")%>"></el-query>
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
						<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>' width="300"></el-table-column>
						<el-table-column prop='rollback_version' label='Rollback Version' width="200"></el-table-column>
						<el-table-column prop='software_version' label='<%=rb.getString("RuanJianBanBen")%>' width="200"></el-table-column>
						<el-table-column prop='module_type' label='<%=rb.getString("SheBeiXingHao")%>' width="200"></el-table-column>
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
										<el-form-item class='deviceItem' label='Rollback Version' prop='rollback_version'>
											<el-select v-model="query_cell_form.rollback_version" size="mini">
												<el-option v-for="item in rbVersionOptions" :key="item.value" :label="item.text" :value="item.value">
												</el-option>
											</el-select>
										</el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("BanBen")%>' prop='software_version'>
											<el-select v-model="query_cell_form.software_version" size="mini">
												<el-option v-for="item in versionOptions" :key="item.value" :label="item.text" :value="item.value">
												</el-option>
											</el-select>
										</el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("SheBeiXingHao")%>' prop='module_type'>
											<el-input v-model='query_cell_form.module_type'  size="mini"></el-input>
										</el-form-item>
									</template>
								</el-query>
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
							<el-query type="normal"  @query="queryEU"  placeholder="'<%=rb.getString("BUJiZhanBianMa")%>/<%=rb.getString("EUJiZhanBianMa")%>'"></el-query>
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
							<el-query type="normal"  @query="queryRU"  placeholder="'<%=rb.getString("RUJiZhanBianMa")%>/<%=rb.getString("RUJiZhanBianMa")%>'"></el-query>
						</template>
					</el-ctable>
				</div>
			</div>
			
			<el-form-item prop='cellCodes' style='margin-bottom:20px;'>
				<el-input v-model='ruleForm.cellCodes' v-show="false"></el-input>
			</el-form-item>
			<div>
				<el-form-item prop='rawMode' style='margin-bottom:5px;' label="<%=rb.getString("WenJianLieBiao") %>" label-width="70px">
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
				<el-radio label="active"><%=rb.getString("LiJiZhiXing")%></el-radio>
				<el-radio label="suspend" style='margin-left:90px;'><%=rb.getString("GuaQi")%></el-radio>
				<el-radio label="timing" style='margin-bottom:0px;margin-left:90px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
			</el-radio-group>
		</el-form-item>
		<el-form-item prop='exetime' style='display:inline-block;vertical-align:bottom;margin-left:15px;margin-bottom:25px;' class='timeItem'>
			<el-date-picker value-format="yyyy-MM-dd HH:mm:ss" v-model='ruleForm.exetime' :disabled="setTimeEnable" type="datetime" @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
		</el-form-item>
	</el-form>
</div>

<script type="text/javascript">
var addUpgrade = new Vue({
	el:'#addCauTask',
	data(){
		var vm = this;
		var validateName = (rule,value,callback) => {
			if(value === ''){
				callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
			}else if(value.trim() == vm.defaultTaskName){
				callback();
			}else{
				axios.post('${ctx}/cau/upgrade/taskNameExist.action',stringify({
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
			fileUrl:'${ctx}/cell/version/queryfileInfosList.action',
			fileParams:{
				timeZone:timeZone,
				productValue:'',
				isShowSlave:false,
				file_type:'cau'
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
				searchText:'',
			},
			query_ru_params:{
				searchText:'',
			},
			query_cell_params:{
				group_id:'',
				serial_number:'',
				host_name:'',
				software_version:'',
				search_text:'',
				module_type:'',
				productValue:'',
				rollback_version:''
			},
			query_cell_form:{
				group_id:'',
				serial_number:'',
				host_name:'',
				software_version:'',
				module_type:'',
				rollback_version:''
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
			fileData:[]
		}
	},
	methods:{ 
		init:function(){
			var vm = this;
			var reg = new RegExp('\\\\',"g");
			vm.productOptions = cauFileVue.buttonGroups;
			var product = isJumpToPage ? isJumpToPage.product : cauFileVue.product_type;
			var productFmt = product.includes("CR-B4860") ? product.substr(0,8) : product.replace(reg,'');
			var fileTypeObj = {
					"CR-B4860/EU" : "7",
					"CR-B4860/BU" : "10",
					"CR-B4860/RU ": "8"
			}
			if(cauFileVue.operType == "modifyTask" ||　cauFileVue.operType == "viewTask"){
				axios.post('${ctx}/task/upgrade/getTask.action',stringify({
					taskId : cauFileVue.rowDataTask.TASK_ID,
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
					vm.fileParams.productValue = productFmt;
					if(vm.deviceType == 'select' && vm.ruleForm.product != 'CR-B4860/EU' && vm.ruleForm.product != 'CR-B4860/RU'){
						vm.query_cell_params.productValue = productFmt;
					}
					vm.$nextTick(function(){
						
						vm.deviceUrl = '${ctx}/cau/upgrade/queryCellInfos.action';
						vm.leftUrl = '${ctx}/cau/upgrade/queryCellInfos.action';
						vm.$refs.ruleForm.clearValidate();
						vm.rightUrl = '${ctx}/cau/upgrade/getTaskSelectedList.action?taskId=' + cauFileVue.rowDataTask.TASK_ID;
						vm.euLeftUrl = '${ctx}/cell/nxp/queryAllEUInfos.action';
						vm.euRightUrl = '${ctx}/cell/nxp/querySelectedEUInfos.action?type=modify&taskId=' + cauFileVue.rowDataTask.TASK_ID ;
						vm.ruLeftUrl = '${ctx}/cell/nxp/queryAllRUInfos.action';
						vm.ruRightUrl = '${ctx}/cell/nxp/querySelectedRUInfos.action?type=modify&taskId=' + cauFileVue.rowDataTask.TASK_ID ;
						initForm(vm.$refs.ruleForm);
    		    	});
					if(cauFileVue.operType == "viewTask"){
						vm.showName = true;
						vm.fileUrl = ''
						axios.post('${ctx}/cell/version/queryfileInfosList.action',stringify({
							timeZone:timeZone,
							productValue:vm.fileParams.productValue,
							isShowSlave:false,
							fileId:vm.ruleForm.file,
							file_type:'cau',
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
						"6" : "6"
				}
				vm.ruleForm.product = product;
				if(vm.ruleForm.product != 'CR-B4860/EU' && vm.ruleForm.product != 'CR-B4860/RU'){
					vm.query_cell_params.productValue = productFmt;
					vm.$nextTick(function(){
						if(!isJumpToPage) vm.$refs.add_task_table.appendCheckedRows(cauFileVue.cellData);
					});
					
				}else{
					if(vm.ruleForm.product == 'CR-B4860/EU'){
						vm.$nextTick(function(){
							if(!isJumpToPage) vm.$refs.add_eu_table.appendCheckedRows(cauFileVue.cellData);
						});
						
					}else{
						vm.$nextTick(function(){
							if(!isJumpToPage) vm.$refs.add_ru_table.appendCheckedRows(cauFileVue.cellData);
						});
					}
				}
			
				vm.fileParams.productValue = productFmt;
				vm.$nextTick(function(){
					vm.deviceUrl = '${ctx}/cau/upgrade/queryCellInfos.action';
					vm.leftUrl = '${ctx}/cau/upgrade/queryCellInfos.action';
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
	    			if(cauFileVue.operType == "modifyTask"){
	    				url = '${ctx}/cau/upgrade/updateTask.action'
	    	    		params.taskId = cauFileVue.rowDataTask.TASK_ID
    	    			if(!isFormChanged(vm.$refs.ruleForm)){
    						vm.$alert('<%=rb.getString("CanShuZhiMeiYouBianHua")%>','<%=rb.getString("TiShi")%>',{
    							confirmButtonText:'<%=rb.getString("QueDing")%>',
    							type:'warning'
    						}).then().catch();
    						return;
       	    			}
	    	    		vm.saveTask(url,params,message);
	    			}else{
	    				url = '${ctx}/cau/upgrade/addTask.action'
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
			axios.post(url,stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
					vm.$message({
						message:message,
						type:'success',
					})
                    cauFileVue.$refs.slide.hide();
                    cauFileVue.list_name = "software";
                    cauFileVue.$refs.upgrade_cell_table.clearSelection();
                    cauFileVue.$refs.upgrade_task_table.refresh();
                    isJumpToPage = '';
				}else{
					vm.$message.error(data["message"])
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
					cauFileVue.$refs.slide.hide();
					isJumpToPage = '';
				}).catch(() => {})
			}else{
				cauFileVue.$refs.slide.hide();
				isJumpToPage = '';
			}
		},
		setTime(){
			this.ruleForm.exetime = formatDate(new Date(gloableTime));
			this.$refs.ruleForm.validateField('exetime');
		},
		loadSuccessFile(data){
			var vm = this;
			if(cauFileVue.operType == "modifyTask" || cauFileVue.operType == "viewTask"){
				axios.post('${ctx}/cau/upgrade/getTask.action',stringify({
					taskId : cauFileVue.rowDataTask.TASK_ID,
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
					vm.query_cell_params.productValue = val.includes("CR-B4860") ? val.substr(0,8) : val.replace(reg,'');
				}else{
					if(val == 'CR-B4860/EU'){
						vm.$nextTick(function(){
							vm.$refs.add_eu_table.clear();
						});
					}else{
						vm.$nextTick(function(){
							vm.$refs.add_ru_table.clear();
						});
					}
				}
			}
			vm.getQueryData(val);
		},
		getQueryData(product){
			var vm = this;
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : product,
				selectType : 'deviceGroup'
			})).then(function(response){
				let data = response.data
				vm.groupOptions = data;
			}).catch(function(error){})
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : product,
				selectType : 'version'
			})).then(function(response){
				let data = response.data
				vm.versionOptions = data;
			}).catch(function(error){})
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				productValue : product,
				selectType : 'rollbackVersion'
			})).then(function(response){
				let data = response.data
				vm.rbVersionOptions = data;
			}).catch(function(error){})
		}
	},
	watch:{
		rowData(newVal){
			this.ruleForm.file = newVal.id
		},
		"ruleForm.status":function(newVal){
			if(cauFileVue.operType =='viewTask' || newVal != 'timing'){
				this.setTimeEnable = true;
			}else{
				this.setTimeEnable = false;
			}
			this.$refs.ruleForm.validateField('exetime')
		},
		"ruleForm.taskType":function(val){
			var vm = this,
				reg = new RegExp('\\\\',"g");
			if(cauFileVue.operType != "viewTask"){
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