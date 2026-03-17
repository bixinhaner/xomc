<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style>
#gnbUpgradeAddTask .modeItem .el-radio{
	display:block;
	margin-left:0px;
	margin-bottom:20px;
} 
#gnbUpgradeAddTask .modeItem{
	margin-top:20px;
}
#gnbUpgradeAddTask .tableDiv{
	height:15px;
	width:auto;
}
#gnbUpgradeAddTask .tableSelectCls{
	cursor: pointer;
	text-align: center;
}
#gnbUpgradeAddTask .tableSelectCls .el-icon::before{
	color: #1EBC1E;
}
#gnbUpgradeAddTask .el-select .el-input.is-disabled .el-input__inner{
	min-height:26px;
	max-height:26px;
}
#gnbUpgradeAddTask .deviceItem{
	display:inline-block;
	margin-right:60px;
}
#gnbUpgradeAddTask .alarmBottomLine{
	background-color:#E9E9E9;
	width: 100%;
	height: 1px;
	margin-bottom: 30px; 
}
#gnbUpgradeAddTask .titleStyML{
	margin-left: 20px;
}
#gnbUpgradeAddTask .deviceTableBox{
	margin:10px 0px 0px 45px;
}
#gnbUpgradeAddTask .gnbDeviceTableBox{
	width: 100%;
	font-size: 12px !important;
}
#gnbUpgradeAddTask .gnbDeviceTableBox .pairgrid-right{
	top:40px!important;
	height: calc(100% - 40px)!important;
}
#gnbUpgradeAddTask .gnbDeviceTableBox .el-pairgrid-title{
	top:15px!important;
	right: 15px!important;
}
#gnbUpgradeAddTask .gnbDeviceTableBox .transition-box .el-form-item{
	display: inline-block;
	margin-right: 30px;
}
#gnbUpgradeAddTask .tableTitles{
	margin-left: 45px;
	margin-top: 20px;
	font-size: 14px;
}
#gnbUpgradeAddTask  .editButton{
	position: absolute;
	top: 15px;
	right: 140px;
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
#gnbUpgradeAddTask .editButton i{
	font-size:14px !important;
}
#gnbUpgradeAddTask .editButton span{
	font-size:12px;
}
.dialogStyle .el-icon-circle-info:before{
	color:#CFCFCF;
}
#gnbUpgradeAddTask .pairgrid-right .el-ctable-toolbar{
	padding: 10px!important;
}
#gnbUpgradeAddTask .el-form-item__label{
	line-height: 26px;
}
#gnbUpgradeAddTask .el-form-item__content{
	margin-left: unset!important;
	width:100%;
}
#gnbUpgradeAddTask .el-input__inner{
	height: 28px;
	line-height: 28px;
}
</style>

<!--新建gnb升级任务 -->
<div id="gnbUpgradeAddTask">
	<el-form :model='ruleForm' :rules="rules" ref="ruleForm" style='padding-top:20px;' label-position="left" label-width="165px"  :hide-required-asterisk=true>
		<div class="group-title not-extend titleStyML" >
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item label='<%=rb.getString("RenWuMingCheng")%>'  prop='taskName' style='margin-left:45px;margin-top:10px;'>
			<el-input maxlength=100 :disabled='viewFlag' v-model="ruleForm.taskName" size="mini" style="width:680px;height:28px;line-height:28px;"></el-input>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div>
			<div class="group-title not-extend titleStyML" >
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
			</div>
			<el-form-item prop="productValue" style='margin:10px 0px 20px 45px;' label="<%=rb.getString("ChangPinXingHao")%>">
				<el-select v-model="ruleForm.productValue" @change="productChange">
					<el-option v-for="item in buttonGroups" :label="item.name" :value="item.value" :key="item.value"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="<%=rb.getString("ZaiXianZhuangTai")%>" prop="connection_status" style="margin:0px 0 20px 45px;" v-show='!viewFlag && false'>
				<el-select v-model="ruleForm.connection_status" @change="connectionStatusChange">
					<el-option v-for="item in onlineOptions" :key="item.value" :label="item.label" :value="item.value"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label='<%=rb.getString("SheBeiZhiDing")%>' style="margin:0px 0px 20px 45px;">
				<el-radio-group  v-model="ruleForm.selectAll" :disabled="viewFlag" @change="selectAllChange">
					<el-radio border  label="true"><%=rb.getString("QuanBu")%></el-radio>
					<el-radio border  label="false"><%=rb.getString("ZhiDingZhiXing")%></el-radio>
				</el-radio-group>
			</el-form-item>
			<div class="deviceTableBox">
				<div class="gnbDeviceTableBox">
					<el-pairgrid 
						:id="'select_device_list'" 
						v-if="showPairGrid && ruleForm.selectAll == 'false'" 
						style="margin-right:45px;" query-name="serial_number" 
						:rownumber="true" 
						ref="gnbDevicePairgrid"  
						@selection-change='selectChange' 
						:right-url="rightUrl" :left-url="leftUrl" 
						:height="height" row-key="small_cell_code" 
						:query-params="queryParams" :title="deviceTitle" 
						:messages="{placeholder:'<%=rb.getString("XiaoZhanBianMa")%>'}"
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
							<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="180"></el-table-column>
							<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>' min-width="180"></el-table-column>
							<el-table-column prop="PHYCELLID" label="<%=rb.getString("PCI2")%>" sortable min-width="80"></el-table-column>
							<el-table-column prop="cell_ip" label="<%=rb.getString("IPDiZhi")%>" sortable min-width="120"></el-table-column>
							<el-table-column prop='rollback_version' label='<%=rb.getString("HuiTuiBanBen")%>' min-width="180"></el-table-column>
							<el-table-column prop='software_version' label='<%=rb.getString("RuanJianBanBen")%>' min-width="180"></el-table-column>
							<el-table-column prop='module_type' label='<%=rb.getString("SheBeiXingHaoMing")%>' min-width="120"></el-table-column>
							<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>' min-width="180"></el-table-column>
						</template>
						<template slot='toolbar'>
							<el-form :model='query_cell_form' ref="query_cell_form" label-position="top">
								<div style="margin:0px 0px 4px 20px;color:#363B4E"><%=rb.getString("SheBeiLieBiao")%></div>

								<div style="display: flex;align-items: center;">
									<el-query type="normal" @query="query" @advance-query="advanceQuery" @reset='resetQuery' :placeholder="'<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>/<%=rb.getString("IPDiZhi")%>'"
									:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
										<!--
										<template slot="form">
											<el-form-item class='deviceItem' label='<%=rb.getString("XiaoZhanBianMa")%>' prop='serial_number'>
												<el-input v-model='query_cell_form.serial_number'  size="mini"></el-input>
											</el-form-item>
											<el-form-item class='deviceItem' label='<%=rb.getString("HostName")%>' prop='host_name'>
												<el-input v-model='query_cell_form.host_name' size="mini"></el-input>
											</el-form-item>
											<el-form-item class='deviceItem' label='<%=rb.getString("IPDiZhi")%>' prop='cell_ip'>
												<el-input v-model='query_cell_form.cell_ip'  size="mini"></el-input>
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
										</template>
										-->
									</el-query>

									<el-popfilter style="margin: 0 5px;" ref="sGroup"
										label="<%=rb.getString("SheBeiZu")%>"
										v-model="query_cell_form.group_id"
										:list="groupOptions.filter(item=>item.value !== '').map(item=>{return {label:item.text,value:item.value}})"
										@check-change="advanceQuery">
									</el-popfilter>
									<el-popfilter style="margin: 0 5px;" ref="sConnection"
										v-show='!viewFlag'
										type="single"
										label="<%=rb.getString("ZaiXianZhuangTai")%>"
										v-model="query_cell_form.connection_status"
										:list="onlineOptions"
										@check-change="advanceQuery">
									</el-popfilter>
									<el-popfilter style="margin: 0 5px;" ref="sRollback"
										label="<%=rb.getString("HuiTuiBanBen")%>"
										v-model="query_cell_form.rollback_version"
										:list="rbVersionOptions.filter(item=>item.value !== '').map(item=>{return {label:item.text,value:item.value}})"
										@check-change="advanceQuery">
									</el-popfilter>
									<el-popfilter style="margin: 0 5px;" ref="sSoftware"
										label="<%=rb.getString("RuanJianBanBen")%>"
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
							<el-table-column prop="serial_number" label="<%=rb.getString("XiaoZhanBianMa")%>" min-width="150"></el-table-column>
						</template>
					</el-pairgrid>
					<el-ctable 
						:id="'all_device_list'" 
						v-if="ruleForm.selectAll == 'true'" 
						:query-params="queryParams" 
						style="border:1px solid #E9E9E9;margin-right:45px;" 
						ref="gnbDevicePairgrid"
						:url="leftUrl" 
						:height="height" 
						pagination="true" 
						>
						<template slot='toolbar'>
							<el-form :model='query_cell_form' ref="query_cell_form" label-position="top" style="display: flex;align-items: center;">
								<el-query type="normal" @query="query" @advance-query="advanceQuery" @reset='resetQuery' :placeholder="'<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>/<%=rb.getString("IPDiZhi")%>'"
								:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
									<!--
									<template slot="form">
										<el-form-item class='deviceItem' label='<%=rb.getString("XiaoZhanBianMa")%>' prop='serial_number'>
											<el-input v-model='query_cell_form.serial_number'  size="mini"></el-input>
										</el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("HostName")%>' prop='host_name'>
											<el-input v-model='query_cell_form.host_name' size="mini"></el-input>
										</el-form-item>
										<el-form-item class='deviceItem' label='<%=rb.getString("IPDiZhi")%>' prop='cell_ip'>
											<el-input v-model='query_cell_form.cell_ip'  size="mini"></el-input>
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
									</template>
									-->
								</el-query>
								
								<el-popfilter style="margin: 0 5px;" ref="aGroup"
									label="<%=rb.getString("SheBeiZu")%>"
									v-model="query_cell_form.group_id"
									:list="groupOptions.filter(item=>item.value !== '').map(item=>{return {label:item.text,value:item.value}})"
									@check-change="advanceQuery">
								</el-popfilter>
								<el-popfilter style="margin: 0 5px;" ref="aConnection"
									v-show='!viewFlag'
									type="single"
									label="<%=rb.getString("ZaiXianZhuangTai")%>"
									v-model="query_cell_form.connection_status"
									:list="onlineOptions"
									@check-change="advanceQuery">
								</el-popfilter>
								<el-popfilter style="margin: 0 5px;" ref="aRollback"
									label="<%=rb.getString("HuiTuiBanBen")%>"
									v-model="query_cell_form.rollback_version"
									:list="rbVersionOptions.filter(item=>item.value !== '').map(item=>{return {label:item.text,value:item.value}})"
									@check-change="advanceQuery">
								</el-popfilter>
								<el-popfilter style="margin: 0 5px;" ref="aSoftware"
									label="<%=rb.getString("BanBen")%>"
									v-model="query_cell_form.software_version"
									:list="versionOptions.filter(item=>item.value !== '').map(item=>{return {label:item.text,value:item.value}})"
									@check-change="advanceQuery">
								</el-popfilter>

								<div class="pop-filter-clear" style="margin: 0 5px;" 
									@click="resetQuery">
									<%=rb.getString("QingKongShaiXuan")%>
								</div>
							</el-form>
						</template>
						<el-table-column prop="connection_status" width="50">
							<template slot-scope="scope">
								<div :class="{
									'el-icon el-icon-status-conn-off':scope.row.connection_status!='Exception' && scope.row.connection_status!='On' && scope.row.connection_status!='updating' && scope.row.connection_status!=1,
									'':scope.row.have_connected==2,
									'conn_exc':scope.row.connection_status=='Exception',
									'el-icon el-icon-status-conn-on':scope.row.connection_status=='On'||scope.row.connection_status=='updating'||scope.row.connection_status==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
							</template>
						</el-table-column>
						<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="180"></el-table-column>
						<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>' min-width="180"></el-table-column>
						<el-table-column prop="PHYCELLID" label="<%=rb.getString("PCI2")%>" sortable min-width="80"></el-table-column>
						<el-table-column prop="cell_ip" label="<%=rb.getString("IPDiZhi")%>" sortable min-width="120"></el-table-column>
						<el-table-column prop='rollback_version' label='<%=rb.getString("HuiTuiBanBen")%>' min-width="180"></el-table-column>
						<el-table-column prop='software_version' label='<%=rb.getString("RuanJianBanBen")%>' min-width="180"></el-table-column>
						<el-table-column prop='module_type' label='<%=rb.getString("SheBeiXingHaoMing")%>' min-width="120"></el-table-column>
						<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>' min-width="180"></el-table-column>
					</el-ctable>
					<div v-if="!showPairGrid && ruleForm.selectAll == 'false'" >
						<el-ctable :id="'selected_device_list'" style="border:1px solid #E9E9E9;margin-right:45px;" ref="stable"  :url="rightUrl" :height="height" front-pagination="true" pagination="true" :query-params="fileParams">
							<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' min-width="180"></el-table-column>
							<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>' min-width="180"></el-table-column>
							<el-table-column prop="PHYCELLID" label="<%=rb.getString("PCI2")%>" sortable min-width="80"></el-table-column>
							<el-table-column prop="cell_ip" label="<%=rb.getString("IPDiZhi")%>" sortable min-width="120"></el-table-column>
							<el-table-column prop='rollback_version' label='<%=rb.getString("HuiTuiBanBen")%>' min-width="180"></el-table-column>
							<el-table-column prop='software_version' label='<%=rb.getString("RuanJianBanBen")%>' min-width="180"></el-table-column>
							<el-table-column prop='module_type' label='<%=rb.getString("SheBeiXingHaoMing")%>' min-width="120"></el-table-column>
							<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>' min-width="180"></el-table-column>
						</el-ctable>
					</div>
				</div>
			</div>
			
			<el-form-item prop='cellCodes' style='margin-left:45px;'>
				<el-input v-model='ruleForm.cellCodes' v-show="false"></el-input>
			</el-form-item>
		</div>
		<div class="tableTitles">
			<el-form-item prop='rawMode' style='margin-bottom:5px;' label="<%=rb.getString("WenJianLieBiao") %>" label-width="80px">
				<el-checkbox :disabled="viewFlag" v-model="ruleForm.rawMode" true-label="false" false-label="true" style='margin-top:3px;'><%=rb.getString("BaoLiuPeiZhi") %></el-checkbox>
			</el-form-item>
		</div>
		<div style='margin:10px 45px 0px 45px;border:1px solid #E9E9E9;'>
			<el-ctable :id="'select_file_list'" row-key="id" :readonly='viewFlag' ref="fileTables" :url='fileDateUrl' :default-checked="defaultChecked" @row-click="rowClickUpgrade" @load-success='loadSuccess' :height="height" pagination="true" :query-params="fileParams">
				<el-table-column label='<%=rb.getString("XuanZe")%>' width="60" >
					<template slot-scope="scope" >
	              		<div class="tableSelectCls">
							<span class='tableDiv el-icon el-icon-status-yes selected-status'></span>
						</div>
	            	</template>
				</el-table-column>
				<el-table-column prop="id" v-if="false"></el-table-column>
				<el-table-column prop="version" show-overflow-tooltip label="<%=rb.getString("BanBen")%>" width="250"></el-table-column>
				<el-table-column prop="product" show-overflow-tooltip label="<%=rb.getString("ChanPinLeiXingBiaoZhi")%>" width="120"></el-table-column>
				<el-table-column prop="file_name" show-overflow-tooltip label="<%=rb.getString("WenJianMing")%>"  width="300"></el-table-column>
				<el-table-column prop="size" label="<%=rb.getString("WenJianDaXiao")%>" width="150"></el-table-column>
				<el-table-column prop="upload_time" label="<%=rb.getString("ShangChuanShiJian")%>" width="180"></el-table-column>
				<el-table-column prop="description" show-overflow-tooltip label="<%=rb.getString("MiaoShu")%>"></el-table-column>
			</el-ctable>
		</div>
		<el-form-item prop='fileId' style='margin-left:45px;'>
			<el-input v-model='ruleForm.fileId' v-show="false"></el-input>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<div style='height:60px;border:none;margin-left:30px;margin-top:20px;display:flex;'>
			<el-form-item  prop='status'>
				<el-radio-group v-model="ruleForm.status" :disabled='viewFlag'>
					<el-radio border label="active" style='margin-left:20px;'><%=rb.getString("LiJiZhiXing")%></el-radio>
					<el-radio border label="suspend"><%=rb.getString("GuaQi")%></el-radio>
					<!--
					<el-radio border label="online"><%=rb.getString("ShangXianZhiXing")%></el-radio>
					-->
					<el-radio border label="timing" style='margin-bottom:0px;'><%=rb.getString("DingShiZhiXing")%></el-radio>
				</el-radio-group>
			</el-form-item>
			<el-form-item prop='exetime' style='display:inline-block;vertical-align:bottom;margin-left:15px;' >
				<el-date-picker style='vertical-align:middle;' value-format="yyyy-MM-dd HH:mm:ss" v-model='ruleForm.exetime' :disabled="viewFlag||setTimeEnable" type="datetime" @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
			</el-form-item>
		</div>
		<!-- 设备并发数 -->
		<div class="group-title not-extend" style='margin-left:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("RenWuShuXing")%></span>
		</div>
		<el-form-item style='display:inline-block;margin:20px 45px 10px;width:50%;' label="<%=rb.getString("LiXianSheBei")%>">
			<el-checkbox v-model="ruleForm.isOnlineExecute" true-label="1" false-label="0" :disabled="viewFlag"></el-checkbox> <%=rb.getString("DengDaiShangXianChongShi")%>
		</el-form-item>
		<el-form-item style='margin:20px 45px;' :label-width="maxNumLabelWidth" prop='maxConcurrentNumber' label="<%=rb.getString("SheBeiBingFaShu")%>">
			<el-input-number @change="maxConcurrentNumberChange"  v-model='ruleForm.maxConcurrentNumber' :min="5" :max="100" :disabled="viewFlag" style="width:200px;height:28px;line-height:28px;"></el-input-number>
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
new Vue({
	el:'#gnbUpgradeAddTask',
	data(){
		var vm = this;
		var validateName = (rule,value,callback) => {
			if(value === ''){
				callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
			}else if(value.trim() == vm.defaultTaskName){
				callback();
			}else{
				axios.post('${ctx}/task/upgrade/taskNameExist.action',stringify({
					taskName:vm.ruleForm.taskName.trim(),
					isGnb:1,
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
		};
		var validateCellCodes = (rule,value,callback) => {
			if(this.ruleForm.selectAll == 'true'){
				callback()
			}else{
				if(value === '' || value === null || value === undefined){
					callback(new Error('<%=rb.getString("QingXuanZeSheBei")%>'))
				}else{
					callback();
				}
			}
		};
		var validatorNum = (rule,value,callback) => {
			var serialNumber = value, temp = /^(\d|[a-zA-Z]|-|\s){1,45}$/,
				list = serialNumber.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});
			
			if (serialNumber == null || serialNumber.length == 0) {
				callback(new Error('<%=rb.getString("SNBuNengWeiKong")%>'));
			}else {
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
			deviceData:{}, // 新建任务接收的设备信息
			leftUrl:'',
			rightUrl:'',
			height:'370px',
			fileParams:{
				timeZone:timeZone,
				productValue:'',
				isGnb:1,
			},
			deviceTitle:['','<%=rb.getString("YiXuan")%>'],
			deviceSelect:[],
			setTimeEnable:true,
			rowData : [],
			selection:'',
			ruleForm:{
				taskName:'${addTaskName}',
				taskType:1,
				cellCodes:'',
				fileId:'',
				fileName:'',
				status:'active',
				exetime:'',
				selectAll:'false',
				version:'',
				maxConcurrentNumber:20,
				productValue:'',
				connection_status: '',
				rawMode:'false',
				isOnlineExecute: '0'
			},
			rules:{
				taskName:[
					{validator:validateName,trigger:'blur'}
				],
				cellCodes:[
					{validator:validateCellCodes,trigger:'change'}
				],
				fileId:[
					{required:true,message:'<%=rb.getString("QingXianXuanZeWenJian")%>',trigger:'change'}
				],
				exetime:[
					{type:'date',validator:validateTime,trigger:'change'}
				]
			},
			defaultChecked:[],
			queryParams:{
				search_text:'',
				timeZone:timeZone,
				isGnb:1,
				like_fields: 'serial_number,host_name,cell_ip',
				group_id:'',
				serial_number:'',
				host_name:'',
				software_version:'',
				productValue:'',
				cell_ip:'',
				rollback_version:"",
				connection_status: ''
			},
			query_cell_form:{
				group_id: [],
				serial_number:'',
				host_name:'',
				software_version: [],
				cell_ip:'',
				rollback_version: [],
				connection_status: ''
			},
			groupOptions:[],
			versionOptions:[],
			rbVersionOptions:[],
			taskId:'',
			operationType:'',
			defaultTaskName:'',
			showPairGrid:true,
			fileDateUrl:'',
			viewFlag:false,
			pickerOptions:{
				disabledDate(time){
					return time.getTime()< Date.getNow()-8.64e7;
				}
			},
			addListForm:{
				serialNumber:''
			},
			addListRules:{
				serialNumber:[
					{validator:validatorNum,trigger:'change'}
				]
			},
			listVisible:false,
			dialogTitle:'<%=rb.getString("TianJia")%>',
			buttonGroups:[],
			clearTableFlag:false,
			onlineOptions:[
				{label:"<%=rb.getString("QuanBu")%>",value:""},
				{label:"<%=rb.getString("ZaiXian")%>",value:"1"},
				{label:"<%=rb.getString("LiXian")%>",value:"0"}
			],
		}
	},
	computed:{
		maxNumLabelWidth(){
			return isLocalZH == true ? '200px' : '290px';
		},
	},
	methods:{ 
		/**
		 * 初始化
		 * @param id:number     gnb升级任务id
		 * @param type:string   任务类型  'addTask' -- 新增  'viewTask'-- 查看  'modifyTask' -- 修改
		*/
		init(id,type){ 
			var vm = this;
			vm.taskId = id ;
			vm.operationType = type;
			
			axios.post("${ctx}/task/upgrade/getProductType.action?isGnb=1").then(function(res){
            	var data = res.data;
				
		   		vm.buttonGroups = data;
			});
			
			if(type == 'viewTask'){
				vm.viewFlag = true;
				vm.showPairGrid = false;
			}else if(type == 'addTask'){
				vm.ruleForm.productValue = gnbFileVue.productValue;
				if(gnbFileVue.cellData.length > 0){
					vm.$refs.gnbDevicePairgrid.appendCheckedRows(gnbFileVue.cellData);
				}
			}
			vm.leftUrl= '${ctx}/task/upgrade/queryCellInfos.action?forSelect=1';
			setTimeout(function(){
				vm.commonSelection();
			},500)
			if(type !== "addTask"){
				vm.$nextTick(function(){
					vm.rightUrl='${ctx}/task/upgrade/getTaskSelectedList.action?taskId=' +vm.taskId;
				})
				vm.getTaskDateInfo();
			}else{
				
				vm.fileDateUrl = '${ctx}/cell/version/queryfileInfosList.action?file_type=0';
			}
			initForm(vm.$refs.ruleForm);
		},
		// 设备执行类别  1 全部执行 2 指定执行
		selectAllChange(val){
			var vm = this;

			vm.resetQuery();
			if(val == 'true'){
				vm.rightUrl = '';
			}else{
				vm.rightUrl='${ctx}/task/upgrade/getTaskSelectedList.action?taskId='+vm.taskId;
			}
			
		},
		commonSelection(){
			var vm = this,
				productVal = vm.ruleForm.productValue.replaceAll('\\','');
			
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				isGnb : 1,
				productValue : productVal,
				selectType : 'deviceGroup'
			})).then(function(response){
				let data = response.data
				vm.groupOptions = data || [];
			}).catch(function(error){})
			
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				isGnb : 1,
				productValue : productVal,
				selectType : 'version'
			})).then(function(response){
				let data = response.data
				vm.versionOptions = data || [];
			}).catch(function(error){})
			
			axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
				isGnb : 1,
				productValue : productVal,
				selectType : 'rollbackVersion'
			})).then(function(response){
				let data = response.data
				vm.rbVersionOptions = data || [];
			}).catch(function(error){})	

			vm.query_cell_form.group_id = [];
			vm.query_cell_form.connection_status = '';
			vm.query_cell_form.rollback_version = [];
			vm.query_cell_form.rollback_version = [];

			['sGroup','sConnection','sRollback','sSoftware','aGroup','aConnection','aRollback','aSoftware'].map(function(code){
				if(vm.$refs[code]) vm.$refs[code].reset();
			})
		},
		// 模糊查询
		query(val){
			var vm = this;
			vm.advanceQuery();
			vm.queryParams.search_text = val;
		},
		// 高级查询
		advanceQuery(){
			var vm = this;
			//vm.queryParams.search_text = "";
			Object.assign(vm.queryParams, vm.query_cell_form);
			vm.queryParams.group_id = vm.queryParams.group_id.join(',');
			vm.queryParams.software_version = vm.queryParams.software_version.join(',');
			vm.queryParams.rollback_version = vm.queryParams.rollback_version.join(',');
		},
		// 重置
		resetQuery(){
			var vm = this,
				params = {
					group_id:'',
					serial_number:'',
					host_name:'',
					software_version:'',
					cell_ip:'',
					rollback_version:'',
					connection_status: ''
				};
			Object.assign(vm.query_cell_form, params, {
				group_id: [],
				software_version: [],
				rollback_version: []
			});
			Object.assign(vm.queryParams, params);
		},
		// 批量输入
		addBatchSn(){
			var vm = this;
			vm.listVisible = true;
		},
		//取消批量输入
		closeBatchSn(){
			var vm = this;
			vm.listVisible = false;
			vm.$refs.addListForm.resetFields();
		},
		saveBatchSn(){
			var vm = this, snStr = vm.addListForm.serialNumber || '';
			list = snStr.replace(/[(\r\n)\r\n\s；]+/g,';').split(';').filter(function(item){ return item.length > 0;});
			var params = {
					serialNumbers : list.join(";"),
					productType : vm.ruleForm.productValue
			}
			vm.$refs.addListForm.validate((valid) => {
				if(valid){
					axios.post("${ctx}/cell/cpeinfos/getTaskCheckSNList.action",stringify(params)).then((res)=>{
						var data = res.data;
						if(data && data.length > 0){		
							vm.$refs.gnbDevicePairgrid.appendCheckedRows(data);
							vm.closeBatchSn();
						}else{
							vm.$message('<%=rb.getString("MeiYouKePiPeiSheBei")%>')
						}
					})
				}
			})
		},
		// gnb设备选择事件
		selectChange(selection){
			var vm = this;
			vm.selection = selection;
		},
		// 升级文件选择事件
		rowClickUpgrade(row){
			var vm = this;
	        vm.rowData = row;
	    },
		// 获取gnb升级任务详情
		getTaskDateInfo(){
			var vm = this,
				params={
					taskId:vm.taskId,
					timeZone:timeZone
				};
			
			axios.post('${ctx}/task/upgrade/getTask.action',stringify(params)).then(function(response){
				let data = response.data;
				vm.ruleForm.taskName = data.TASK_NAME;
				vm.ruleForm.fileId = data.FILE_ID;
				vm.ruleForm.status = data.CREATE_STATUS;
				vm.ruleForm.productValue = data.PRODUCT_TYPE;
				vm.ruleForm.maxConcurrentNumber = data.maxConcurrentNumber;
				vm.ruleForm.isOnlineExecute = data.isOnlineExecute;
				
				if(data.CREATE_STATUS == 'timing'){
					vm.ruleForm.exetime = data.CREATE_TIME;
				}else{
					vm.ruleForm.exetime = '';
				}
				vm.ruleForm.rawMode = data.IS_KEEP_CONFIG;
				vm.defaultChecked = [data.FILE_ID+''];
				vm.defaultTaskName = data.TASK_NAME;
				vm.ruleForm.selectAll = data.selectAll;
				if(data.selectAll == ''){
					vm.ruleForm.selectAll = 'false';
				}
				vm.fileDateUrl = '${ctx}/cell/version/queryfileInfosList.action'+"&fileId="+data.fileId+"&timeZone="+timeZone;
				
			}).catch(function(error){})
		},
		// 提交
		submit(){
	    	var vm = this,
                message = '<%=rb.getString("ChengGong")%>';
            // 防止多次提交
			if(gnbFileVue.slideSubmitLoading)return

			if(vm.operationType == 'modifyTask'){
				if(!isFormChanged(vm.$refs.ruleForm)){
					vm.$alert('<%=rb.getString("CanShuZhiMeiYouBianHua")%>','<%=rb.getString("TiShi")%>',{
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						type:'warning'
					}).then().catch();
					return;
				}
			}
	    	vm.$refs.ruleForm.validate((valid) => {
	    		if(valid){
	    			var params = {},urls='';
	    			params.timeZone = timeZone;
					params.isGnb = 1;
	    			params.cellCodes = vm.ruleForm.cellCodes;
	    			params.taskName = vm.ruleForm.taskName;
					params.selectAll = vm.ruleForm.selectAll;
	    			params.status = vm.ruleForm.status;
					params.taskType = vm.ruleForm.taskType;
					params.productValue = vm.ruleForm.productValue;
					params.maxConcurrentNumber = vm.ruleForm.maxConcurrentNumber;
					params.isOnlineExecute = vm.ruleForm.isOnlineExecute;
					params.rawMode = vm.ruleForm.rawMode;
	    			if(vm.ruleForm.status == 'timing'){
	    				params.time = vm.ruleForm.exetime;
	    			}
	    			params.fileId = vm.ruleForm.fileId;
					if(vm.operationType == 'addTask'){
						urls = '${ctx}/task/upgrade/addTask.action';
					}else{
						params.taskId = vm.taskId
						urls = '${ctx}/task/upgrade/updateTask.action';
					}
					vm.saveTask(urls,params,message);
	    		}else{
	    			return false;
	    		}
	    	})
		},
		saveTask(urls,params,message){
			var vm = this;
            gnbFileVue.slideSubmitLoading = true;
			axios.post(urls,stringify(params)).then(function(response){
				var data = response.data;
				if(data["success"]){
					vm.$message({
						message:message,
						type:'success',
					})
                    gnbFileVue.$refs.upgrade_cell_table.clearSelection();
                    gnbFileVue.$refs.slide.hide();
                    gnbFileVue.list_name = "software";
                    isJumpToPage = '';
				}else{
					vm.$message.error(data["message"]);
                    gnbFileVue.slideSubmitLoading = false;
				}
			}).catch(function(error){})
			
		},
		// 取消
		cancel(){
			var vm = this;
			var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>';
			if(vm.operationType !== 'taskView'){
				if(isFormChanged(this.$refs.ruleForm)){
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						customClass:'warningConfirm',
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						gnbFileVue.$refs.slide.hide();
						isJumpToPage = ''
					}).catch(() => {
						
					})
				}else{
					gnbFileVue.$refs.slide.hide();
					isJumpToPage = ''
				}
			}else{
				gnbFileVue.$refs.slide.hide();
				isJumpToPage = ''
			}
			
		},
		// gnb 升级文件表格加载成功回调
		loadSuccess(){
			var vm = this;
			var tb = vm.$refs.fileTables;
			var rows = tb.tbData;
			if(vm.operationType !== 'addTask'){
				vm.updateRow(tb.tbData,tb);
			}else{
				if(isJumpToPage){
					rows.map(function(item){
						if(item.id == isJumpToPage.vid){
							tb.setCurrentRow(item)
						}
					})
				}
			}
			
		},
		updateRow(rows,tb){
			var vm = this;
			axios.post('${ctx}/task/upgrade/getTask.action',stringify({
				taskId : vm.taskId,
				timeZone : timeZone
			})).then(function(response){
				var data = response.data;
				rows.map(function(item){
					if(item.id == data.FILE_ID){
						tb.setCurrentRow(item)
					}
				})
			})
		},
		// 定时时间失焦事件
		setTime(){
			var vm = this;
			vm.ruleForm.exetime = formatDate(new Date(gloableTime));
			vm.$refs.ruleForm.validateField('exetime');
		},
		// 产品类型改变
		productChange(val){
			var vm = this;
			vm.commonSelection();
		},
		connectionStatusChange(val){
			var vm = this;
			vm.queryParams.connection_status = val;			
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
		}
	},
	watch:{
		rowData(newVal){
			var vm = this;

			this.ruleForm.fileId = newVal.id;
			this.ruleForm.fileName = newVal.file_name;
			this.ruleForm.version = newVal.version;
		},
		"ruleForm.status":function(newVal){
			if(newVal == 'timing'){
				this.setTimeEnable = false
			}else{
				this.setTimeEnable = true
			}
			this.$refs.ruleForm.validateField('exetime')
		},
		selection(){
			var vm = this,
				data = this.$refs.gnbDevicePairgrid.getData();
				cellCodes = '',cellCodeList = [];
			if(data.length != 0){
				data.map(function(item){
					cellCodeList.push(item.small_cell_code);
				})
			}
			cellCodes = cellCodeList.join(',');
			vm.ruleForm.cellCodes = cellCodes;
		},
		"ruleForm.productValue":function(newVal){
			this.queryParams.productValue = newVal;
			this.fileParams.productValue = newVal;
			if(this.clearTableFlag == true){
				this.$refs.gnbDevicePairgrid.clear();
			}
			this.clearTableFlag = true;
		},
	},
	mounted(){
		eventBus.$off('task-init').$on('task-init',this.init);
		eventBus.$off('add-task').$on('add-task',this.submit);
		eventBus.$off('cancel-add-task').$on('cancel-add-task',this.cancel);
	}
})
</script>