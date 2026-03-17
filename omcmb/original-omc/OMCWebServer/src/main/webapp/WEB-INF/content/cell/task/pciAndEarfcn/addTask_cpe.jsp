<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>
<style>
	
	#addPciTaskCpe .errorBorder .el-input__inner{
		border:1px solid red;
	}
	.el-popover{
		font-size:12px;
	}
	.el-popover--plain{
		padding:10px !important
	}
	#addPciTaskCpe .errorMsg .el-form-item__error{
		left:30px;
	}
	
	#addPciTaskCpe .alarmBottomLine{
		background-color:#E9E9E9;
		width: 100%;
		height: 1px;
		margin-bottom: 30px; 
	}
	#addPciTaskCpe .titleStyML{
		margin-left: 20px;
	}
	
	#addPciTaskCpe .activeStatusItem .el-icon-status-active:before{
		color:#67D972;
	}
	#addPciTaskCpe .inactiveStatusItem .el-icon-status-active:before{
		color:#E88282;
	}
	.gnb-label-flex .el-form-item__label {
		text-align: left;
		line-height: 28px;
	}
	.gnb-half-item {
		display: flex;
		flex-wrap: wrap;
	}
	.gnb-half-item .el-form-item {
		flex: 1 1 40%;
		margin-right: 40px;
	}
	#addPciTaskCpe .el-form-item{
		margin-bottom: 20px;
	}
</style>
<div id='addPciTaskCpe' style="margin-top:20px;">
	<el-form ref='cpePciForm' :model="cpePciForm" :rules="rules" label-position="left">
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("JiBenXinXi")%></span>
		</div>
		<el-form-item label='<%=rb.getString("RenWuMingCheng")%>' style='margin-left:45px;margin-top:20px;' prop='taskName' label-width="120px">
			<el-input v-model='cpePciForm.taskName' :disabled='viewCpe' maxlength=50 size="mini" style="width:400px;height:28px;line-height:28px;padding-top:5px;"></el-input>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML">
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("SheBeiXuanZe")%></span>
		</div>
		<el-form-item prop='selectModel' style='margin-left:45px;margin-top:20px;' label='<%=rb.getString("SheBeiXingHao")%>' label-width="120px">
			<el-radio-group v-model="cpePciForm.selectModel" @change="cpeModelChange">
				<el-radio label="4G" border size="small">4G CPE</el-radio>
				<el-radio label="5G" border size="small">5G CPE</el-radio>
			</el-radio-group>
		</el-form-item>
		<el-pairgrid v-if='showDevice' style='margin:0px 45px 0px 45px;' :id="'pci_select_list'" ref="cpairgrid" @selection-change='selectChange'  :right-url="rightUrl" :left-url="leftUrl" :height="height" row-key="CPE_CODE" :query-params="queryForm" :title="deviceTitle" :messages="{placeholder:'<%=rb.getString("CPEBianMa")%>'}" @right-load-success="rightLoadSuccess">
			<template slot="left">
				<el-table-column type='selection' :reserve-selection="true" width="40"></el-table-column>
				<el-table-column prop="CONNECTION_STATUS" width="50">
					<template slot-scope="scope">
						<div :class="{
							'el-icon el-icon-status-conn-off':scope.row.CONNECTION_STATUS!='Exception' && scope.row.CONNECTION_STATUS!='On' && scope.row.CONNECTION_STATUS!='updating' && scope.row.CONNECTION_STATUS!=1,
							'':scope.row.have_connected==2,
							'conn_exc':scope.row.CONNECTION_STATUS=='Exception',
							'el-icon el-icon-status-conn-on':scope.row.CONNECTION_STATUS=='On'||scope.row.CONNECTION_STATUS=='updating'||scope.row.CONNECTION_STATUS==1 || ['initializing','syncSourceInSync','syncSourceInSynced'].includes(scope.row.connection_status) }" style='font-size:22px;'></div>
					</template>
				</el-table-column>
				<el-table-column prop='CPE_CODE' v-if=false></el-table-column>
				<el-table-column prop='SERIAL_NUMBER' label='<%=rb.getString("CPEBianMa")%>' min-width="200"></el-table-column>
				<el-table-column prop='CPE_NAME' label='<%=rb.getString("CPEName")%>' min-width='150'></el-table-column>
				<el-table-column label='<%=rb.getString("SheBeiXingHao")%>' prop="cpe_model" key="cpe_model"  min-width="120"></el-table-column>
				<el-table-column prop='EARFCN' label='<%=rb.getString("PinDian")%>' min-width='100'></el-table-column>
				<el-table-column prop='PCI' label='PCI' min-width='100'>
					<template slot-scope='scope'>
						<div style="display: flex;" v-html="pciStatus(scope.row.PCI,scope.row)"></div>
					</template>
				</el-table-column>
				<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>' min-width='200'></el-table-column>
			</template>
			<template slot='toolbar'>
				<el-query @query="query" @advance-query="advanceQuery" @reset='resetQuery' :placeholder="'<%=rb.getString("CPEBianMa")%>/<%=rb.getString("CPEName")%>/PCI'"
				:ok-text="'<%=rb.getString("ChaXun")%>'" :reset-text="'<%=rb.getString("ChaXunChongZhi")%>'">
					<template slot="form">
						<div class='queryInfo'>
							<label><%=rb.getString("CPEBianMa")%></label>
							<el-input v-model='queryForm.serial_number'  size="mini"></el-input>
						</div>
						<div class='queryInfo'>
							<label><%=rb.getString("CPEName")%></label>
							<el-input v-model='queryForm.CPE_NAME' size="mini"></el-input>
						</div>
						<div class='queryInfo'>
							<label><%=rb.getString("SheBeiZu")%></label>
							<el-select v-model="queryForm.group_id" size="mini">
								<el-option v-for="item in groupOptions" :key="item.value" :label="item.text" :value="item.value">
								</el-option>
							</el-select>
						</div>
						<div class='queryInfo'>
							<label><%=rb.getString("PinDian")%></label>
							<el-input v-model='queryForm.earfcn'  size="mini"></el-input>
						</div>
						<div class='queryInfo'>
							<label>PCI</label>
							<el-input v-model='queryForm.pci'  size="mini"></el-input>
						</div>
					</template>
				</el-query>
			</template>
			<template slot='right'>
				<el-table-column prop='SERIAL_NUMBER' label='<%=rb.getString("CPEBianMa")%>' width="200"></el-table-column>
				<el-table-column prop='CPE_NAME' label='<%=rb.getString("CPEName")%>' width='100'></el-table-column>
				<el-table-column prop='EARFCN' label='<%=rb.getString("PinDian")%>' width='100'></el-table-column>
				<el-table-column prop='PCI' label='PCI' width='100'>
					<template slot-scope='scope'>
						<div v-if='scope.row.PCI == null||scope.row.PCI.toString().indexOf("_")==-1'>
							<span v-html='scope.row.PCI'></span>
						</div>
						<div v-else>
							<p v-if="scope.row.PCI.toString().substring(0,1) == 1">
								<span class='el-icon el-icon-operation-lock'></span><span style='margin-left:5px;' v-html='scope.row.PCI.toString().substring(2)'></span>
							</p>
							<p v-if="scope.row.PCI.toString().substring(0,1) == 0">
								<span class='el-icon el-icon-status-unlock'></span><span v-html='scope.row.PCI.toString().substring(2)'></span>
							</p>
						</div>
					</template>
				</el-table-column>
				<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>' width='200'></el-table-column>
			</template>
		</el-pairgrid>
		<div v-else style='margin:20px 45px 0px 45px;'>
			<el-ctable  ref="selected_table" :rownumber="true" id="selected_table" :url="rightUrl" height="360px"  :pagination="false" style="border:1px solid #E9E9E9;">
				<el-table-column label='<%=rb.getString("CPEBianMa")%>' min-width="200" prop='SERIAL_NUMBER'></el-table-column>
				<el-table-column label='<%=rb.getString("CPEName")%>' min-width="200" prop="CPE_NAME" show-overflow-tooltip="true"></el-table-column>
				<el-table-column label='<%=rb.getString("PinDian")%>' min-width="100" prop="EARFCN"></el-table-column>
				<el-table-column label='PCI' min-width="80" prop="PCI"></el-table-column>
				<el-table-column label='<%=rb.getString("SheBeiZu")%>' min-width="200" prop="group_name"></el-table-column>
			</el-ctable>
		</div>
		
		<el-form-item prop='cpeCodes' class='errorMsg' style='margin-left:17px;'>
			<el-input v-model='cpePciForm.cpeCodes' v-show=false></el-input>
		</el-form-item>
		<div class="alarmBottomLine"></div>
		<div v-show="cpePciForm.selectModel == '4G'">
			<div class="group-title not-extend titleStyML">
				<span class="title-icon"></span>
				<span class="title-text"><%=rb.getString("PinDianSuo")%></span>
			</div>
			<div style='margin-left:30px;position:relative;border:none;width:80%;margin-top:20px;'>
				<div style='width:900px;display:inline-block;vertical-align:middle'>
					<el-form-item prop='lock_mode'>
						<el-radio-group :disabled='viewCpe' style='margin-top:20px;' v-model='cpePciForm.lock_mode' @change='changeScanMode'>
							<el-radio label='fullband' style='margin-right:60px;margin-left:20px;'>Full Band</el-radio>
							<el-radio label='freqpreferred' style='margin-right:60px;'>Frequency Preferred</el-radio>
							<el-radio label='pcilock' style='margin-right:60px;'>PCI Lock</el-radio>
							<el-radio label='pcionlylock'>PCI Only Lock</el-radio>
						</el-radio-group>
					</el-form-item>
				</div>
				<!-- Frequency Preferred -->
				<div v-if='showEarfcn' style='margin-left:20px;'>
					<div>
						<label style='display:inline-block;font-size:14px;margin:10px 10px 5px 0px;width:65px;'>Earfcn</label>
						<el-input v-model='frequencyItem' :disabled='viewCpe' :class="{errorBorder:isFrequencyError}" style='width:300px;'></el-input>
						<span class='form-bt el-icon el-icon-plus' :class='addClass' @click='addEarfcn' style='vertical-align:top'></span>
						<span class='scan_icon' style='font-size:20px;vertical-align:middle;margin-left:20px;' @click="selectVal"></span>
						<span style='color:#7F7F7F;margin-left:5px;'><%=rb.getString("SuoPinZuiDuoXuanZe3Ge")%></span>
					</div>
					<div>
						<div v-for='(domain,index) in cpePciForm.earfcnGroup' style='margin-left:80px;margin-top:5px;'>
							<p style='width:100px;height:24px;border:1px solid #4D84FF;line-height:24px;padding-left:10px;position:relative;background:#F2F6FF;display:inline-block'>
								<span>{{domain.earfcn}}</span>
								<span v-show="showClose" class='el-icon el-icon-close' style='font-size:16px;right:10px;top:5px;position:absolute;' @click.prevent='removeDomain(domain.earfcn)'></span>
							</p>
							<div style='display:inline-block;margin-left:10px;' v-if="domain.sn.length > 0">
								<span v-show="domain.sn.length > 0">SN:{{domain.sn[0]}}</span>
								<span v-show="domain.sn.length > 1">...</span>
								<el-popover placement="bottom" width="200" trigger="click">
									<div>
										<p v-for="item in domain.sn" style='margin-bottom:5px;'>SN:{{item}}</p>
									</div>
									<span slot="reference" v-show="domain.sn.length > 1" style='color:#7584FF;cursor:pointer'>[{{domain.sn.length}}]</span>
								</el-popover>
								
							</div>
							<div v-else style='display:inline-block;margin-left:10px;'>No match enbs</div>
						</div>
						<el-form-item prop='earfcnGroupStr' v-show=false>
							<el-input v-model='cpePciForm.earfcnGroupStr'></el-input>
						</el-form-item>
					</div>
					<p style='color:red;'>{{errorMessage}}</p>
					<el-form-item prop='itemTest'>
						<el-input v-model='cpePciForm.itemTest' v-show=false></el-input>
					</el-form-item>
				</div>

				<!-- PCI Lock -->
				<div v-if='showPci' style='margin-left:20px;'>
					<div>
						<label style='display:inline-block;font-size:14px;margin:10px 10px 5px 0px;width:105px;'>Earfcn - PCI</label>
						<div style='display:inline-block'>
							<el-input v-model='earfcnItem' :disabled='viewCpe' :class="{errorBorder:isEarfcnError}" style='width:300px;'></el-input>&nbsp;&nbsp;-&nbsp;&nbsp;<el-input v-model='pciItem' :disabled='viewCpe' :class="{errorBorder:isPciError}" style='width:300px;'></el-input>
						</div>
						<span class='form-bt el-icon el-icon-plus' :class='addCpeClass' @click='addPci' style='vertical-align:top'></span>
						<span class='scan_icon' style='font-size:20px;vertical-align:middle;margin-left:20px;' @click="selectVal"></span>
						<span style='color:#7F7F7F;margin-left:5px;'><%=rb.getString("SuoPinZuiDuoXuanZe3Ge")%></span>
					</div>
					<div>
						<div v-for='(domain,index) in cpePciForm.pciGroup' style='margin-left:120px;margin-top:5px;'>
							<p style='width:200px;height:24px;border:1px solid #4D84FF;line-height:24px;padding-left:10px;position:relative;background:#F2F6FF;display:inline-block'>
								<span>Earfcn:&nbsp;{{domain.value.split("_")[0]}}&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;PCI:&nbsp;{{domain.value.split("_")[1]}}</span>
								<span v-show="showClose" class='el-icon el-icon-close' style='font-size:16px;right:10px;top:5px;position:absolute;' @click.prevent='removePci(domain.value)'></span>
							</p>
							<div style='display:inline-block;margin-left:10px;' v-if="domain.sn.length > 0">
								<span v-show="domain.sn.length > 0">SN:{{domain.sn[0]}}</span>
								<span v-show="domain.sn.length > 1">...</span>
								<el-popover placement="bottom" width="200" trigger="click">
									<div>
										<p v-for="item in domain.sn" style='margin-bottom:5px;'>SN:{{item}}</p>
									</div>
									<span slot="reference" v-show="domain.sn.length > 1" style='color:#7584FF;cursor:pointer'>[{{domain.sn.length}}]</span>
								</el-popover>
							</div>
							<div v-else style='display:inline-block;margin-left:10px;'>No match enbs</div>
						</div>
						<el-form-item prop='pciGroupStr' v-show=false>
							<el-input v-model='cpePciForm.pciGroupStr'></el-input>
						</el-form-item>
					</div>
					<p style='color:red;'>{{pciErrMessage}}</p>
					<el-form-item prop='itemTest'>
						<el-input v-model='cpePciForm.itemTest' v-show=false></el-input>
					</el-form-item>
				</div>

				<!-- PCI Only Lock -->
				<div v-if='showPcionlylock' style='margin-left:20px;'>
					<div>
						<label style='display:inline-block;font-size:14px;margin:10px 10px 5px 0px;width:30px;'>PCI</label>
						<el-input v-model='pcionlylockItem' :disabled='viewCpe' style='width:300px;'></el-input>
						<span class='form-bt el-icon el-icon-plus' :class='addOnlyClass' @click='addPcionlylock' style='vertical-align:top'></span>
						<span class='scan_icon' style='font-size:20px;vertical-align:middle;margin-left:20px;' @click="selectVal"></span>
						<span style='color:#7F7F7F;margin-left:5px;'><%=rb.getString("SuoPinZuiDuoXuanZe3Ge")%></span>
					</div>
					<div>
						<div v-for='(domain,index) in cpePciForm.pcionlylockGroup' style='margin-left:44px;margin-top:5px;'>
							<p style='width:100px;height:24px;border:1px solid #4D84FF;line-height:24px;padding-left:10px;position:relative;background:#F2F6FF;display:inline-block'>
								<span>{{domain.pci}}</span>
								<span v-show="showClose" class='el-icon el-icon-close' style='font-size:16px;right:10px;top:5px;position:absolute;' @click.prevent='removePcionlylock(domain.pci)'></span>
							</p>
							<div style='display:inline-block' v-if="domain.sn.length > 0">
								<span v-show="domain.sn.length > 0">SN:{{domain.sn[0]}}</span>
								<span v-show="domain.sn.length > 1">...</span>
								<el-popover placement="bottom" width="200" trigger="click">
									<div>
										<p v-for="item in domain.sn" style='margin-bottom:5px;'>SN:{{item}}</p>
									</div>
									<span slot="reference" v-show="domain.sn.length > 1" style='color:#7584FF;cursor:pointer'>[{{domain.sn.length}}]</span>
								</el-popover>
							</div>
							<div v-else style='display:inline-block;margin-left:10px;'>No match enbs</div>
						</div>
						<el-form-item prop='pcionlylockGroupStr' v-show=false>
							<el-input v-model='cpePciForm.pcionlylockGroupStr'></el-input>
						</el-form-item>
					</div>
					<p style='color:red;'>{{errorMessage}}</p>
					<el-form-item prop='itemTest'>
						<el-input v-model='cpePciForm.itemTest' v-show=false></el-input>
					</el-form-item>
				</div>
			</div>
		</div>
		<div v-show="cpePciForm.selectModel == '5G'" >
			<!-- 5G -->
			<div class="group-title not-extend titleStyML">
				<span class="title-icon"></span>
				<span class="title-text">5G <%=rb.getString("PinDianSuo")%></span>
			</div>
			<div style='margin-left:60px;position:relative;border:none;width:95%;margin-top:20px;'>
				<el-form-item v-show="false" prop="nrLockMode">
					<el-input v-model="cpePciForm.nrLockMode"></el-input>
				</el-form-item>
				<el-form-item v-show="false" prop="nrFreq">
					<el-input v-model="cpePciForm.nrFreq"></el-input>
				</el-form-item>
				<el-form-item v-show="false" prop="nrPci">
					<el-input v-model="cpePciForm.nrPci"></el-input>
				</el-form-item>
				<el-form-item v-show="false" prop="nrBand">
					<el-input v-model="cpePciForm.nrBand"></el-input>
				</el-form-item>

				<el-form-item label="<%=rb.getString("SaoMiaoFangShi")%>" prop="nrLockMode" label-width="120px">
					<el-select :disabled='viewCpe' v-model="cpePciForm.nrLockMode" style="height:28px;line-height:28px;padding-top:5px;">
						<el-option label="Full Band" value="fullband"></el-option>
						<el-option label="Frequency Lock" value="freqlock"></el-option>
						<el-option label="Cell Lock" value="celllock"></el-option>
						<el-option label="Band Lock" value="bandlock"></el-option>
					</el-select>
				</el-form-item>
				<!-- 5G Frequency lock -->
				<div v-show="cpePciForm.nrLockMode=='freqlock'">
					<el-form-item label="Frequency Lock" style="margin-bottom: 5px;">
						<i v-if="!viewCpe" class="el-icon el-icon-plus" style="margin-top: 10px;" @click="showFreqLock"></i>
					</el-form-item>
					<el-table height="200px" :data="freqLockList">
						<el-table-column label="Index" type="index" width="100"></el-table-column>
						<el-table-column label="Rat" prop="rat">
							<template slot-scope="scope">
								<span v-if="scope.row.rat == '0'">4G LTE</span>
								<span v-if="scope.row.rat == '1'">5G NR</span>
							</template>
						</el-table-column>
						<el-table-column label="Band" prop="band"></el-table-column>
						<el-table-column label="Freq" prop="freq"></el-table-column>
						<el-table-column v-if="!viewCpe" label="Operation" width="100">
							<template slot-scope="scope">
								<i @click="deleteFreqLock(scope.$index)" class="el-icon el-icon-operation-delete"></i>
							</template>
						</el-table-column>
					</el-table>
				</div>
				<!-- 5G cell lock -->
				<div v-show="cpePciForm.nrLockMode=='celllock'">
					<el-form-item label="Cell Lock" style="margin-bottom: 5px;">
						<i v-if="!viewCpe" class="el-icon el-icon-plus" style="margin-top: 10px;" @click="showCellLock"></i>
					</el-form-item>
					<el-table height="200px" :data="cellLockList">
						<el-table-column label="Index" type="index" width="100"></el-table-column>
						<el-table-column label="Rat" prop="rat">
							<template slot-scope="scope">
								<span v-if="scope.row.rat == '0'">LTE</span>
								<span v-if="scope.row.rat == '1'">NR</span>
							</template>
						</el-table-column>
						<el-table-column label="Band" prop="band"></el-table-column>
						<el-table-column label="Earfcn" prop="earfcn"></el-table-column>
						<el-table-column label="PCI" prop="pci"></el-table-column>
						<el-table-column v-if="!viewCpe" label="Operation" width="100">
							<template slot-scope="scope">
								<i @click="deleteCellLock(scope.$index)" class="el-icon el-icon-operation-delete"></i>
							</template>
						</el-table-column>
					</el-table>
				</div>

				<!-- 5G Band lock -->
				<div v-show="cpePciForm.nrLockMode=='bandlock'">
					<el-form-item label="5G Band Lock" style="margin-bottom: 5px;">
						<i v-if="!viewCpe" class="el-icon el-icon-plus" style="margin-top: 10px;" @click="showBandLock"></i>
					</el-form-item>
					<el-table height="200px" :data="bandLockList">
						<el-table-column label="Index" type="index" width="100"></el-table-column>
						<el-table-column label="Rat" prop="rat">
							<template slot-scope="scope">
								<span v-if="scope.row.rat == '0'">LTE</span>
								<span v-if="scope.row.rat == '1'">SA</span>
								<span v-if="scope.row.rat == '2'">NSA</span>
							</template>
						</el-table-column>
						<el-table-column label="Band" prop="band"></el-table-column>
						<el-table-column v-if="!viewCpe" label="Operation" width="100">
							<template slot-scope="scope">
								<i @click="deleteBandLock(scope.$index)" class="el-icon el-icon-operation-delete"></i>
							</template>
						</el-table-column>
					</el-table>
				</div>
			</div>
		</div>
		<div class="alarmBottomLine"></div>
		<div class="group-title not-extend titleStyML" style='margin-top:20px;'>
			<span class="title-icon"></span>
			<span class="title-text"><%=rb.getString("ZhiXingFangShi")%></span>
		</div>
		<div style='height:60px;border:none;margin-left:30px;margin-top:20px;display:flex;'>
			<el-form-item prop='status'>
				<el-radio-group v-model='cpePciForm.status' :disabled='viewCpe' style='margin-top:17px;' @change="statusChange">
					<el-radio label='active' style='margin-right:110px;margin-left:20px;'><%=rb.getString("LiJiZhiXing")%></el-radio>
					<el-radio label='suspend' style='margin-right:110px;'><%=rb.getString("GuaQi")%></el-radio>
					<el-radio label='timing'><%=rb.getString("DingShiZhiXing")%></el-radio>
				</el-radio-group>
			</el-form-item>
			<el-form-item prop='exetime' class='errorMsg'>
				<el-date-picker v-model="cpePciForm.exetime"  style='margin-top:10px;vertical-align:middle;margin-left:25px;' :disabled='setTimeEnable' value-format="yyyy-MM-dd HH:mm:ss" type="datetime"  @focus='setTime' :picker-options="pickerOptions"></el-date-picker>	
			</el-form-item>
		</div>
	</el-form>
	<el-dialog @close="closeDialog" title='Earfcn/PCI List' :visible.sync='dialogVisible' width='50%' top='25vh' :close-on-click-modal = false append-to-body = true>
	 	<div>
	 		<el-ctable ref="pciListTable" :id="'pciListTable'" :url='pciListUrl' :height="listHeight"  :query-params="enb_pci_params" pagination="true" @selection-change='selectList' row-key="small_cell_code">
				<template slot="toolbar">
					<div class='queryGroup' style='height:28px;'>
						<el-input v-model='enb_pci_form.search_text' @keyup.enter.native="enbPciQuery" class='pairgrid-query' placeholder='<%=rb.getString("XiaoZhanBianMa")%>/<%=rb.getString("HostName")%>/<%=rb.getString("PinDian")%>/PCI'></el-input>
						<i @click='enbPciQuery' class="el-icon el-icon-common-search" style="margin-left: 10px;"></i>
					</div>
				</template>
				<el-table-column type='selection' width='55'></el-table-column>
				<el-table-column prop='serial_number' label='<%=rb.getString("XiaoZhanBianMa")%>' width='200'></el-table-column>
				<el-table-column prop='host_name' label='<%=rb.getString("HostName")%>' width='200'></el-table-column>
				<el-table-column prop='EARFCNDLINUSE' label='<%=rb.getString("PinDian")%>' width='80'></el-table-column>
				<el-table-column prop='PHYCELLID' label='PCI' width='80'></el-table-column>
				<el-table-column prop='group_name' label='<%=rb.getString("SheBeiZu")%>' width='150'></el-table-column>
				<el-table-column prop='op_state' label='<%=rb.getString("ShiFouJiHuo")%>' :formatter = "statusFmt">
					<template slot-scope="scope">
						<div style='display:flex' v-html="statusFmt(scope.row.op_state)"></div>
					</template>
				</el-table-column>
			</el-ctable>
	 	</div>
	 	<el-button-group style='margin-left:10px;margin-bottom:10px;' slot="footer">
			<el-button type='primary' size='small' @click='savePciList'><%=rb.getString("QueDing")%></el-button>
			<el-button size='small' @click='closeDialog'><%=rb.getString("QuXiao")%></el-button>
		</el-button-group>
	 </el-dialog>

	<el-dialog title="Add 5G Frequency Lock" :visible.sync="freq5gDlShow" :modal="true" :append-to-body="true" :close-on-click-modal="false">
		<el-form ref="freqForm" :model="freqForm" :rules="lockRules" label-width="80" class="gnb-label-flex gnb-half-item">
			<el-form-item label="Rat" prop="rat">
				<el-select v-model="freqForm.rat" @change="freqLockRatChange">
					<el-option label="4G LTE" value="0"></el-option>
					<el-option label="5G NR" value="1"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="Band" prop="band">
				<el-select v-model="freqForm.band" @change="freqLockBandChange" filterable>
					<el-option v-for="item in bandList" :label="item?item:'Full'" :value="item"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="Freq" prop="freq" required>
				<el-input v-model="freqForm.freq"></el-input>
			</el-form-item>
		</el-form>
		
		<div slot="footer">
			<el-button type="primary" @click="addFreqLock"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="freq5gDlShow = false"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>

	<el-dialog title="Add 5G Cell Lock" :visible.sync="cell5gDlShow" :modal="true" :append-to-body="true" :close-on-click-modal="false">
		<el-form ref="cellForm" :model="cellForm" :rules="lockRules" label-width="80" class="gnb-label-flex gnb-half-item">
			<el-form-item label="Rat" prop="rat">
				<el-select v-model="cellForm.rat">
					<el-option label="LTE" value="0"></el-option>
					<el-option label="NR" value="1"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="Band" prop="band">
				<el-select v-model="cellForm.band" filterable>
					<el-option v-for="item in bandList" :label="item?item:'Full'" :value="item"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="Earfcn" prop="earfcn" required>
				<el-input v-model="cellForm.earfcn"></el-input>
			</el-form-item>
			<el-form-item label="PCI" prop="pci" required>
				<el-input v-model="cellForm.pci"></el-input>
			</el-form-item>
		</el-form>
		
		<div slot="footer">
			<el-button type="primary" @click="addCellLock"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="cell5gDlShow = false"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
		
	<el-dialog title="Add 5G Band Lock" :visible.sync="band5gDlShow" :modal="true" :append-to-body="true" :close-on-click-modal="false">
		<el-form ref="bandForm" :model="bandForm" :rules="lockRules" label-width="80" class="gnb-label-flex gnb-half-item">
			<el-form-item label="Rat" prop="rat">
				<el-select v-model="bandForm.rat">
					<el-option label="LTE" value="0"></el-option>
					<el-option label="SA" value="1"></el-option>
					<el-option label="NSA" value="2"></el-option>
				</el-select>
			</el-form-item>
			<el-form-item label="Band" prop="band">
				<el-select v-model="bandForm.band" filterable>
					<el-option v-for="item in bandList" :label="item?item:'Full'" :value="item"></el-option>
				</el-select>
			</el-form-item>
		</el-form>
		
		<div slot="footer">
			<el-button type="primary" @click="addBandLock"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="band5gDlShow = false"><%=rb.getString("QuXiao")%></el-button>
		</div>
	</el-dialog>
</div>

<script>
	var addCpePci = new Vue({
		el:'#addPciTaskCpe',
		data(){
			var vm = this;
			var validateName = (rule,value,callback) => {
				if(value === ''){
					callback(new Error('<%=rb.getString("QingShuRuXinJianRenWuMingCheng")%>'))
				}else if(value.trim() == vm.defaultTaskName){
					callback();
				}else{
					axios.post('${ctx}/cpe/strategy/taskNameExist.action',stringify({
						taskName:vm.cpePciForm.taskName.trim()
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
				if(this.cpePciForm.status !== 'timing'){
					callback()
				}else{
					if(value == '' || value==null){
						callback(new Error('<%=rb.getString("QingXuanZeShiJian")%>'))
					}else{
						callback();
					}
				}
			};
			var validateLock = (rule,value,callback) => {
				if(this.cpePciForm.lock_mode == 'freqpreferred'){
					if(this.cpePciForm.earfcnGroup.length == 0){
						callback(new Error('<%=rb.getString("ZhiShaoTianJiaYiGe")%>'))
					}else if(this.cpePciForm.earfcnGroup.length > 3){
						callback(new Error('No more than three'))
					}else{
						callback()
					}
				}
				if(this.cpePciForm.lock_mode == 'pcilock'){
					var flag = true;
					vm.cpePciForm.pciGroup.map(function(item){
						var earfcn = item.value.split("_")[0].toString();
						var pci = item.value.split("_")[1].toString();
						if(earfcn == "" || pci == ""){
							flag = false;
						}
					})
					if(this.cpePciForm.pciGroup.length == 0){
						callback(new Error('<%=rb.getString("ZhiShaoTianJiaYiGe")%>'))
					}else if(this.cpePciForm.pciGroup.length > 3){
						callback(new Error('No more than three'))
					}else if(flag == false){
						flag = true;
						callback(new Error('<%=rb.getString("ZhiShaoTianJiaYiGe")%>'))
					}else{
						callback()
					}
				}
				if(this.cpePciForm.lock_mode == 'pcionlylock'){
					if(this.cpePciForm.pcionlylockGroup.length == 0){
						callback(new Error('<%=rb.getString("ZhiShaoTianJiaYiGe")%>'))
					}else if(this.cpePciForm.pcionlylockGroup.length > 3){
						callback(new Error('No more than three'))
					}else{
						callback()
					}
				}
			};
			var validateEarfcn = function(rule,value,cb) {
					if(value !== '') {
						if(value - 0 < 0 || value - 3279156 > 0 || isNaN(value)) {
							cb('<%=rb.getString("5GPinDianFanWei")%>');
						}else {
							cb();
						}
					}else {
						cb('Please Input Earfcn');
					}
				},
				
				validatePCI = function(rule,value,cb) {
					if(value !== '') {
						if(value - 0 < 0 || value - 1007 > 0 || isNaN(value)) {
							cb('<%=rb.getString("5GSpecificPCIFanWei")%>');
						}else {
							cb();
						}
					}else {
						cb('Please Input PCI');
					}
				},
				validateFreq = function(rule,value,callback) {
					var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/,
						bandVal = vm.freqForm.band,
						ratType = vm.freqForm.rat,
						rangeStr = ratType == '0' ? vm.bandCounterpartFreq4GList[bandVal] : vm.bandCounterpartFreq5GList[bandVal],
						minVal = parseInt(rangeStr.split('~')[0]),
						maxVal = parseInt(rangeStr.split('~')[1]);
	
					if(value == '' || value == undefined || value == null) {
						callback('<%=rb.getString("FanWei")%>：'+ rangeStr +',Integer');
					}else {
						if(reg.test(value) && value >= minVal && value <= maxVal) {
							callback();
						}else {
							callback('<%=rb.getString("FanWei")%>：'+ rangeStr +',Integer');
						}
					}
				};

			return{
				freqLockList: [],
				cellLockList: [],
				bandLockList: [],
				freq5gDlShow: false,
				cell5gDlShow: false,
				band5gDlShow: false,
				freqForm: {
					rat: '0',
					band: '1',
					freq: '',
				},
				cellForm: {
					rat: '',
					band: '',
					earfcn: '',
					pci: ''
				},
				bandForm: {
					rat: '',
					band: ''
				},
				lockRules: {
					rat: [{required: true, message: 'Please Select Rat'}],
					band: [{required: true, message: 'Please Select Band'}],
					earfcn: [{validator: validateEarfcn}],
					pci: [{validator: validatePCI}],
					freq: [{validator: validateFreq}]
				},

				cpePciForm:{
					taskName:'${taskName}',
					status:'active',
					exetime:'',
					earfcnGroupStr:'',
					pciGroupStr:'',
					lock_mode:'fullband',
					cpeCodes:'',
					itemTest:'',
					pcionlylockStr:'',
					earfcnGroup:[],
					pciGroup:[],
					pcionlylockGroup:[],

					nrLockMode: 'fullband',
					nrFreq:'',
					nrPci: '',
					nrBand: '',

					selectModel:'4G'
				},
				rules:{
					taskName:[
						{validator:validateName,trigger:'blur'}
					],
					cpeCodes:[
						{required:true,message:'<%=rb.getString("QingXuanZeSheBei")%>',trigger:'change'}
					],
					exetime:[
						{type:'date',validator:validateTime,trigger:'change'}
					],
					itemTest:[
						{validator:validateLock}
					]
				},
				leftUrl:'${ctx}/cell/CPE/queryCpeInfosList.action?forSelect=9',
				rightUrl:'',
				queryForm:{
					serial_number:'',
					CPE_NAME:'',
					group_id:'',
					earfcn:'',
					pci:'',
					search_text:'',
					like_fields:'serial_number,cpe_name,pci',
					cpe_model:'4G'
				},
				height:'360px',
				deviceTitle:['<%=rb.getString("CPELieBiao")%>','<%=rb.getString("YiXuan")%>'],
				groupOptions:[],
				pickerOptions:{
					disabledDate(time){
						return time.getTime()< Date.now()-8.64e7;
					}
				},
				selection:[],
				showEarfcn:false,
				showPci:false,
				showPcionlylock:false,
				isFrequencyError:false,
				frequencyItem:'',
				errorMessage:'',
				pciErrMessage:'',
				addClass:'',
				earfcnItem:'',
				pciItem:'',
				addCpeClass:'',
				addOnlyClass:'',
				isEarfcnError:false,
				isPciError:false,

				setTimeEnable:true,
				defaultTaskName:'',
				taskId:'',
				operType:'',
				viewCpe:false,
				showDevice:true,
				checkUrl:'${ctx}/cell/cpeinfos/queryCpeInfosList.action?lockMode=fullband&forSelect=10&page=1&rows=500',
				isfullband:false,
				pcionlylockItem:'',
				scanEarfcn:'disabled',
				scanPci:'disabled',
				scanPciOnly:'disabled',
				dialogVisible:false,
				pciListUrl:'${ctx}/cell/cpeinfos/queryCpeInfosList.action?forSelect=9',
				pciParams:{
					like_fields:"serial_number,host_name",
					search_text:'',
					CA_FLAG:'0',
					serial_number:'',
					host_name:'',
					group_id:'',
					earfcn:''
				},
				listHeight:'450px',
				pciListSelection:[],
				showClose:true,
				enb_pci_params:{search_text:'',like_fields:'serial_number,host_name,earfcndlinuse,phycellid'},
				enb_pci_form:{search_text:''},
				bandCounterpartFreq4GList:{
					'1':'0~599','2':'600~1199','3':'1200~1949','4':'1950~2399','5':'2400~2649','7':'2750~3449','8':'3450~3799','12':'5010~5179',
					'13':'5180~5279','14':'5280~5379','17':'5730~5849','18':'5850~5999','19':'6000~6149','20':'6150~6449','25':'8040~8689',
					'26':'8690~9039','28':'9210~9659','29':'9660~9769','30':'9770~9869','32':'9920~10359','34':'36200~36349','38':'37750~38249',
					'39':'38250~38649','40':'38650~39649','41':'39650~41589','42':'41590~43589','43':'43590~45589','46':'46790~54539','48':'55240~56739',
					'66':'66436~67335','71':'68586~68935'
				},
				bandCounterpartFreq5GList:{
					'1':'422000~434000','2':'386000~398000','3':'361000~376000','5':'173800~178800','7':'524000~538000','8':'185000~192000',
					'12':'145800~149200','13':'149200~151200','14':'151600~153600','18':'172000~175000','20':'158200~164200','25':'386000~399000',
					'26':'171800~178800','28':'151600~160600','29':'65535~65535','30':'470000~472000','38':'514000~524000','40':'460000~480000',
					'41':'499200~537999','48':'636667~646666','66':'422000~440000','70':'399000~404000','71':'123400~130400','75':'286400~303400',
					'76':'285400~286400','77':'620000~680000','78':'620000~653333','79':'693334~733333'
				},
			}
		},
		computed: {
			bandList() {
				var vm = this,
                	nrLockMode = vm.cpePciForm.nrLockMode;
				let list = [];
				if(nrLockMode == 'celllock' || nrLockMode == 'bandlock'){
					for(let i=0;i<=100;i++) {
						list.push(i);
					}
				}else if(nrLockMode == 'freqlock'){
					var ratType = vm.freqForm.rat;
					if(ratType == '0'){
						list = ['1','2','3','4','5','7','8','12','13','14','17','18','19','20','25','26','28','29','30','32','34','38','39','40','41','42','43','46','48','66','71'];
					}else if(ratType == '1'){
						list = ['1','2','3','5','7','8','12','13','14','18','20','25','26','28','29','30','38','40','41','48','66','70','71','75','76','77','78','79'];
					}
				}
				return list;
			}
		},
		methods:{
			
			// cpe 初始化
			init(id,operType){
				var vm = this;
				
				vm.operType = operType;
				vm.taskId = id;
				vm.getSelectedData();
				if(vm.operType == 'view'){
					vm.viewCpe = true;
					vm.showDevice = false;
					vm.addClass='disabled';
					vm.addCpeClass='disabled';
					vm.addOnlyClass='disabled';
					vm.showClose = false;
					vm.setTimeEnable = true;
				}
				if(vm.operType !== 'add'){
					vm.rightUrl = '${ctx}/cpe/strategy/getSelectedList.action?taskId=' + vm.taskId;
					axios.post('${ctx}/cpe/strategy/getCPEPciLockTaskInfo.action?taskId=' + vm.taskId +'&timeZone=' + timeZone).then(function(response){
						let data = response.data,
							nrlockMode = data["nrLockMode"]?data["nrLockMode"]:'fullband',
							nrPcis = data["nrEarfcnAndPci"]?data["nrEarfcnAndPci"].split(';'):[],
							nrFreqs = data["freqlock"]?data["freqlock"].split(';'):[];

						vm.cpePciForm.taskName = data.taskName;
						vm.defaultTaskName = data.taskName;
						vm.cpePciForm.status = data.executeType;
						
						vm.cpePciForm.selectModel = data.selectModel;
						if(data.executeType == 'timing'){
							vm.cpePciForm.exetime = data.quartzTime;
						}
						if(vm.cpePciForm.selectModel == '4G'){
							vm.cpePciForm.lock_mode = data.lockMode? data.lockMode:'fullband';
							if(data.lockMode == 'freqpreferred'){
								vm.showEarfcn = true;
								vm.isfullband = true;
								
								if(![null,undefined,''].includes(data.earfcnAndPci)) {
									let arr = Array.from(new Set(data.earfcnAndPci.split(',')));
									arr.map(function(item){
										vm.checkEarfcn(item)
									})
								}
							}
							if(data.lockMode == 'pcilock'){
								vm.showPci = true;
								vm.isfullband = true;
								
								if(![null,undefined,''].includes(data.earfcnAndPci)) {
									let arr = Array.from(new Set(data.earfcnAndPci.split(',')));
									arr.map(function(item){
										let index = item.indexOf('_');
										let earfcn = item.substring(0,index);
										let pci = item.substring(index+1);
										vm.checkPci(earfcn,pci);
									})
								}
							}
							if(data.lockMode == 'pcionlylock'){
								vm.showPcionlylock = true;
								vm.isfullband = true;

								if(![null,undefined,''].includes(data.earfcnAndPci)) {
									let arr = Array.from(new Set(data.earfcnAndPci.split(';')));
									arr.map(function(item){
										vm.checkOnlyPci(item)
									})
								}
							}
						}else{
							// 5G
							vm.cpePciForm.nrLockMode = nrlockMode;
							if(nrlockMode == 'celllock') {
								nrPcis.map(function(item){
									let arr = item.split(',');

									vm.cellLockList.push({
										rat: arr[0],
										band: arr[1],
										earfcn: arr[2],
										pci: arr[3]
									});
								});
							}else if(nrlockMode == 'bandlock') {
								nrPcis.map(function(item){
									let arr = item.split(',');

									vm.bandLockList.push({
										rat: arr[0],
										band: arr[1]
									});
								});
							}else if(nrlockMode == 'freqlock') {
								nrFreqs.map(function(item){
									let arr = item.split(',');

									vm.freqLockList.push({
										rat: arr[0],
										band: arr[1],
										freq: arr[2]
									});
								});
							}
						}
					})
					setTimeout(function(){
						initForm(vm.$refs.cpePciForm);
					},2000);
				}else{
					vm.cpePciForm.selectModel = pciLockVue.cpe_model;
					vm.queryForm.cpe_model = vm.cpePciForm.selectModel;
					if(pciLockVue.selectionCpe.length > 0){
						vm.$nextTick(function(){
							vm.$refs.cpairgrid.appendCheckedRows(pciLockVue.selectionCpe);
						})
						
					}
				}
				
				
			},
			getSelectedData(){
				var vm = this;
				axios.post('${ctx}/cell/cpeinfos/getCellSelectFilter.action',stringify({
					selectType : 'deviceGroup'
				})).then(function(response){
					let data = response.data
					vm.groupOptions = data;
				}).catch(function(error){})
			},
			/**
			 *select 选择数据
			 * @param selection:所选数据
			*/
			selectChange(selection){
				this.selection = selection;
			},
			// 模糊查询
			query(val){
				this.resetQuery();
				this.queryForm.search_text = val;
				this.$refs.cpairgrid.reload();
			},
			//基站列表 搜索框 高级查询确定按钮
			advanceQuery(){
				this.queryForm.search_text = "";
				this.$refs.cpairgrid.reload();
			},
			//基站列表 搜索框 高级查询重置按钮
			resetQuery(){
				this.queryForm.serial_number = '';
				this.queryForm.CPE_NAME = '';
				this.queryForm.group_id = '';
				this.queryForm.earfcn = '';
				this.queryForm.pci = '';
			},
			pciStatus(value, rowData) {
				var reg = new RegExp("^(IDU\/CN)");
				if("LTE WiFi VoIP Gateway" == rowData["OLDPRODUCT"] || (reg.test(rowData["OLDPRODUCT"])==true) ){
					return "--";
				}
				if(!value) return "";
				var pciValue = rowData.PCI;
				if(rowData.SCANMODE == 'pcilock' || rowData.SCANMODE == 'pcionlylock' ){
					var imgL = "<div class='pciClass'><span class='el-icon el-icon-operation-lock easyui-tooltip'></span></div><span style='margin-left:5px;'>"+pciValue+"</span>" ;
					return imgL;
				}else{
					var imgL = "<div class='pciClass'><span class='el-icon el-icon-status-unlock easyui-tooltip'></span></div><span>"+pciValue+"</span>" ;
					return imgL;
				}
			},
			setTime(){
				this.cpePciForm.exetime = formatDate(new Date(gloableTime));
				this.$refs.cpePciForm.validateField('exetime');
			},
			// 添加端口号
			addEarfcn(){
				var vm = this;
				if(vm.addClass == 'disabled'){
					return;
				}else{
					var value = vm.frequencyItem.trim();
					if(value == '' || isNaN(value)){
						vm.isFrequencyError = true;
						vm.errorMessage = '<%=rb.getString("PinDianFanWei")%>';
						return;
					}else if(value < 0 || value > 65535){
						vm.isFrequencyError = true;
						vm.errorMessage = '<%=rb.getString("PinDianFanWei")%>';
						return;
					}else{
						var list = vm.cpePciForm.earfcnGroup.map(function(item){
								return item.earfcn
							});

						if(list.includes(value)) {
							vm.isFrequencyError = true;
							vm.errorMessage = '<%=rb.getString("YiCunZai")%>';
						}else {
							vm.checkEarfcn(value);
						}
					}
				}
			},
			
			checkEarfcn(value){
				var vm = this;
				var obj = {
						earfcn : value,
						sn:[]
					};
				var params = {
						lockMode : 'freqpreferred',
						earfcnAndPci : value.toString(),
						forSelect : 10
				}
				axios.post("${ctx}/cell/cpeinfos/queryCpeInfosList.action",stringify(params)).then(function(response){
					var data = response.data;
					if(data.rows.length != 0){
						data.rows.map(function(item){
							obj.sn.push(item.serial_number);
						})
					}
					vm.cpePciForm.earfcnGroup.push(obj);
					vm.isFrequencyError = false;
					vm.errorMessage = '';
					vm.frequencyItem = '';
					if(vm.cpePciForm.earfcnGroup.length == 3){
						vm.addClass = 'disabled'
					}
					vm.$refs.cpePciForm.validateField('itemTest');
				})
			},
			// 添加Pci lock
			addPci(){
				var vm = this;
				if(vm.addCpeClass == 'disabled'){
					return;
				}else{
					var earfcn = vm.earfcnItem.trim();
					var pci = vm.pciItem.trim();
					if(earfcn == '' || isNaN(earfcn) || earfcn < 0 || earfcn > 65535){
						vm.isEarfcnError = true;
						vm.pciErrMessage = '<%=rb.getString("PinDianFanWei")%>';
						return;
					}else{
						vm.isEarfcnError = false;
						vm.pciErrMessage = '';
					}
					if(pci == '' || isNaN(pci) || pci < 0 || pci > 503){
						vm.isPciError = true;
						vm.pciErrMessage = '<%=rb.getString("SpecificPCIFanWei")%>';
						return;
					}else{
						vm.isPciError = false;
						vm.pciErrMessage = '';
					}

					var list = vm.cpePciForm.pciGroup.map(function(item){
								return item.value
							});

					if(list.includes( earfcn + "_" + pci)) {
						vm.isPciError = true;
						vm.pciErrMessage = '<%=rb.getString("YiCunZai")%>';
					}else {
						vm.checkPci(earfcn,pci);
					};
				}
			},
			checkPci(earfcn,pci){
				var vm = this;
				var obj = {
						value : earfcn + "_" +pci,
						sn:[]
				}
				var params = {
						lockMode : 'pcilock',
						earfcnAndPci :earfcn +'_' + pci,
						forSelect : 10
				}
				axios.post("${ctx}/cell/cpeinfos/queryCpeInfosList.action",stringify(params)).then(function(response){
					var data = response.data;
					if(data.rows.length != 0){
						data.rows.map(function(item){
							obj.sn.push(item.serial_number);
						})
					}
					vm.cpePciForm.pciGroup.push(obj);
					vm.earfcnItem = '';
					vm.pciItem = '';
					if(vm.cpePciForm.pciGroup.length == 3){
						vm.addCpeClass='disabled'
					}
					vm.$refs.cpePciForm.validateField('itemTest');
				})
			},
			// 添加Pci only Lock
			addPcionlylock(){
				var vm = this;
				if(vm.addOnlyClass == 'disabled'){
					return;
				}else{
					var value = vm.pcionlylockItem.trim();
					if(value == '' || isNaN(value)){
						vm.errorMessage = '<%=rb.getString("SpecificPCIFanWei")%>';
						return;
					}else if(value < 0 || value > 503){
						vm.errorMessage = '<%=rb.getString("SpecificPCIFanWei")%>';
						return;
					}else{
						var list = vm.cpePciForm.pcionlylockGroup.map(function(item){
								return item.pci
							});

						if(list.includes(value)) {
							vm.errorMessage = '<%=rb.getString("YiCunZai")%>';
						}else {
							vm.checkOnlyPci(value)
						};
					}
				}
			},
			checkOnlyPci(value){
				var vm = this;
				var obj = {
					pci:value,
					sn:[]	
				}
				var params = {
						lockMode : 'pcionlylock',
						earfcnAndPci :value.toString(),
						forSelect : 10
				}
				axios.post("${ctx}/cell/cpeinfos/queryCpeInfosList.action",stringify(params)).then(function(response){
					var data = response.data;
					if(data.rows.length != 0){
						data.rows.map(function(item){
							obj.sn.push(item.serial_number);
						})
					}
					vm.cpePciForm.pcionlylockGroup.push(obj);
					vm.errorMessage = '';
					vm.pcionlylockItem = '';
					if(vm.cpePciForm.pcionlylockGroup.length == 3){
						vm.addOnlyClass = 'disabled'
					}
					vm.$refs.cpePciForm.validateField('itemTest');
				})
			},
			// 删除端口号
			removeDomain(item){
				var vm = this;
				var list = vm.cpePciForm.earfcnGroup.map(function(item){
					return item.earfcn;
				})
				var index = list.indexOf(item);
				if(index !== -1){
					vm.cpePciForm.earfcnGroup.splice(index,1)
				}
				vm.addClass = vm.cpePciForm.earfcnGroup.length >= 3 ? 'disabled' : '';
				vm.isFrequencyError = false;
				vm.errorMessage = '';
				vm.$refs.cpePciForm.validateField('itemTest');
			},
			// 删除 Pci lock
			removePci(item){
				var vm = this;
				var list = vm.cpePciForm.pciGroup.map(function(item){
					return item.value;
				})
				var index = list.indexOf(item);
				if(index !== -1){
					this.cpePciForm.pciGroup.splice(index,1)
				}
				vm.addCpeClass = vm.cpePciForm.pciGroup.length >= 3 ? 'disabled' : '';
				vm.isEarfcnError = false;
				vm.isPciError = false;
				vm.pciErrMessage = '';
				vm.$refs.cpePciForm.validateField('itemTest');
			},
			// 删除Pci only Lock
			removePcionlylock(item){
				var vm = this;
				var list = vm.cpePciForm.pcionlylockGroup.map(function(item){
					return item.pci;
				})
				var index = list.indexOf(item);
				if(index !== -1){
					vm.cpePciForm.pcionlylockGroup.splice(index,1)
				}
				vm.addOnlyClass = vm.cpePciForm.pcionlylockGroup.length >= 3 ? 'disabled' : '';
				vm.errorMessage = '';
				vm.$refs.cpePciForm.validateField('itemTest');
			},
			/**
			 * 频点锁 radio 选择
			 * @param label:选择数据
			*/
			changeScanMode(label){
				var vm = this;
				vm.errorMessage = "";
				vm.pciErrMessage = "";
				
				if(label == 'freqpreferred'){
					vm.showEarfcn = true;
					vm.showPci = false;
					vm.isfullband = true;
					vm.showPcionlylock = false;
				}else if(label == 'pcilock'){
					vm.showPci = true;
					vm.showEarfcn = false;
					vm.isfullband = true;
					vm.showPcionlylock = false;
				}else if(label == 'pcionlylock'){
					vm.showEarfcn = false;
					vm.showPci = false;
					vm.isfullband = true;
					vm.showPcionlylock = true;
				}else{
					vm.showEarfcn = false;
					vm.showPci = false;
					vm.isfullband = false;
					vm.showPcionlylock = false;
				}
			},
			// 关闭新建cpe 窗口
			cancelSubmit(){
				var vm = this;
				var confirmStr = '<%=rb.getString("QueDingLiKaiDangQianYeMian")%>'
				if(isFormChanged(vm.$refs.cpePciForm)){
					vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
						customClass:'warningConfirm',
						confirmButtonText:'<%=rb.getString("QueDing")%>',
						cancelButtonText:'<%=rb.getString("QuXiao")%>',
						type:'warning',
						closeOnClickModal:false
					}).then(() => {
						eventBus.$emit('close-cpe');
					}).catch(() => {
						
					})
				}else{
					eventBus.$emit('close-cpe');
				}
			},
			// 执行方式改变事件
			statusChange(val){
				var vm = this;
				if(val !== 'timing'){
					vm.cpePciForm.exetime = '';
					vm.$refs.cpePciForm.clearValidate('exetime')
				}
			},
			// 确定保存按钮
			submit(){
				var vm = this;
                // 防止多次提交
                if(pciLockVue.slideSubmitLoading)return

				vm.$refs.cpePciForm.validate((valid) => {
					if(valid){
						var freqStr = '';
						var pciStr = '';
						var pcionlylockStr = '';
						var saveUrl = '';
						var saveStr = '';
						var params = {
								timeZone : timeZone
							}
						params.selectModel = vm.cpePciForm.selectModel;
						params.taskName = vm.cpePciForm.taskName;
						params.status = vm.cpePciForm.status;
						if(vm.cpePciForm.status == 'timing'){
							params.time = vm.cpePciForm.exetime;
						}
						
						params.cpe_codes = vm.cpePciForm.cpeCodes;
						if(vm.cpePciForm.selectModel == '4G'){
							params.lock_mode = vm.cpePciForm.lock_mode;
							if(vm.cpePciForm.lock_mode == 'freqpreferred'){
								freqStr = vm.cpePciForm.earfcnGroup.map(function(item){
									return item.earfcn;
								})
								params.earfcn_pci = freqStr.toString();
							}
							if(vm.cpePciForm.lock_mode == 'pcilock'){
								pciStr = vm.cpePciForm.pciGroup.map(function(item){
									return item.value;
								})
								params.earfcn_pci = pciStr.toString();
							}
							if(vm.cpePciForm.lock_mode == 'pcionlylock'){
								pcionlylockStr = vm.cpePciForm.pcionlylockGroup.map(function(item){
									return item.pci;
								})
								params.earfcn_pci = pcionlylockStr.toString();
							}
						}else{
							params['nr_lock_mode'] = vm.cpePciForm['nrLockMode'];

							if(vm.cpePciForm.nrLockMode == 'celllock') {
								params['nr_earfcn_pci'] = vm.cpePciForm['nrPci'];
							}else if(vm.cpePciForm.nrLockMode == 'bandlock') {
								params['nr_earfcn_pci'] = vm.cpePciForm['nrBand'];
							}else if(vm.cpePciForm.nrLockMode == 'freqlock'){
								params['nr_earfcn_pci'] = vm.cpePciForm['nrFreq'];
							}
						}

						if(vm.operType == 'add'){
							saveUrl = '${ctx}/cpe/strategy/addTask.action?type=add';
							saveStr = '<%=rb.getString("ChengGong")%>'
						}else if(vm.operType == 'modify'){
							if(isFormChanged(vm.$refs.cpePciForm)){
								saveUrl = '${ctx}/cpe/strategy/addTask.action?type=modify&taskId=' +vm.taskId;
								saveStr = '<%=rb.getString("ChengGong")%>'
							}else{
								var submitStr = '<%=rb.getString("WuCanShuBianHua")%>'
								vm.$alert(submitStr,'<%=rb.getString("QueRen")%>',{
									confirmButtonText:'<%=rb.getString("QueDing")%>'
								})
							}
						}
                        pciLockVue.slideSubmitLoading = true;
						axios.post(saveUrl,stringify(params)).then(function(response){
							let data = response.data;
							if(data["success"]){
								vm.$message({
									message:saveStr,
									type:'success'
								})
								eventBus.$emit('save-cpe');
								pciLockVue.$refs.cpe_list_table.clearSelection();
							}else{
								vm.$message.error(data.message);
                                pciLockVue.slideSubmitLoading = false;
							}
						})
					}else{
						return false;
					}
				})
			},
			
			selectVal(){
				this.dialogVisible = this.operType == 'view' ? false : true
			},
			selectList(selection){
				this.pciListSelection = selection;
			},
			savePciList(){
				var vm = this,
					lockMode = vm.cpePciForm.lock_mode;
				if(lockMode == 'freqpreferred'){
					var list = vm.pciListSelection.map(function(item){
						return item.EARFCNDLINUSE;
					});
					var earfcnList = Array.from(new Set(list));
					earfcnList.map(function(item){
						var obj = {
								earfcn : item,
								sn:[]
						}
						vm.pciListSelection.map(function(item1){
							if(item1.EARFCNDLINUSE == item){
								obj.sn.push(item1.serial_number)
							}
						})
						vm.cpePciForm.earfcnGroup.push(obj);
					})
					if(vm.cpePciForm.earfcnGroup.length >= 3){
						vm.addClass = 'disabled';
					}
				}else if(lockMode == 'pcilock'){
					var list = vm.combinData(vm.pciListSelection);
					var existArr = vm.cpePciForm.pciGroup.map(function(item){
						return item.value
					})
					list.map(function(item){
						if(existArr.includes(item.value)){
							var index = existArr.indexOf(item.value);
							vm.cpePciForm.pciGroup[index].sn = item.sn
						}else{
							vm.cpePciForm.pciGroup.push(item)
						}
					})
					if(vm.cpePciForm.pciGroup.length >= 3){
						vm.addCpeClass = 'disabled';
					}
				}else if(lockMode == 'pcionlylock'){
					var list = vm.pciListSelection.map(function(item){
						return item.PHYCELLID;
					});
					var pcionlyList = Array.from(new Set(list));
					pcionlyList.map(function(item){
						var obj = {
								pci : item,
								sn:[]
						}
						vm.pciListSelection.map(function(item1){
							if(item1.PHYCELLID == item){
								obj.sn.push(item1.serial_number)
							}
						})
						vm.cpePciForm.pcionlylockGroup.push(obj);
					})
					if(vm.cpePciForm.pcionlylockGroup.length >= 3){
						vm.addOnlyClass = 'disabled';
					}
				}
				vm.dialogVisible = false;
				vm.$refs.pciListTable.clearSelection();
				vm.$refs.cpePciForm.validateField('itemTest');
			},
			combinData(rows){
				var map = {};
				var list = [];
				rows.map(function(row){
					var earfcns = (row.EARFCNDLINUSE == null ? '' : row.EARFCNDLINUSE).split(",");
					var pcis = (row.PHYCELLID == null ? '' : row.PHYCELLID).split(",");
					var sn = row.serial_number;
					var minNum = Math.min(earfcns.length,pcis.length);
					for(var i = 0;i<minNum;i++){
						var key = earfcns[i] + '_' + pcis[i];
						if(map[key]){
							if(!map[key].includes(sn)) map[key].push(sn)
						}else{
							map[key] = [sn];
						}
					}
				})
				for(var key in map){
					list.push({value : key,sn:map[key]})
				}
				return list;
			},
			closeDialog(){
				this.dialogVisible = false;
				this.$refs.pciListTable.clearSelection();
			},
			rightLoadSuccess(data){
				var cpeCodes = data.map(function(item){
					return item.CPE_CODE;
				})
				this.cpePciForm.cpeCodes = cpeCodes.toString();
				initForm(this.$refs.cpePciForm);
			},
			enbPciQuery(){
				Object.assign(this.enb_pci_params,this.enb_pci_form);
			},
			statusFmt(value){
				if (value == null) {
					return null;
				}
				else if (value == "1") {
					val = "<%= rb.getString("JiHuo")%>";
					value = "<div class='activeStatusItem'><span class='el-icon el-icon-status-active' style='margin-right: 10px;'></span>"+(val)+"</div>"
				}
				else if (value == "0") {
					//状态不一样展示的文字也不一样
					val = "<%= rb.getString("QuJiHuo")%>";
					value = "<div class='inactiveStatusItem'><span class='el-icon el-icon-status-active' style='margin-right: 10px;'></span>"+(val)+"</div>"
				}
				return value;
			},
			
			showFreqLock() {
				let vm = this;

				vm.resetLockForm();
				vm.freq5gDlShow = true;
				vm.$nextTick(function(){
					vm.$refs.freqForm.clearValidate();
				});
			},
			showCellLock() {
				let vm = this;

				vm.resetLockForm();
				vm.cell5gDlShow = true;
				vm.$nextTick(function(){
					vm.$refs.cellForm.clearValidate();
				});
			},
			showBandLock() {
				let vm = this;

				vm.resetLockForm();
				vm.band5gDlShow = true;
				vm.$nextTick(function(){
					vm.$refs.bandForm.clearValidate();
				});
			},
			resetLockForm() {
				var vm = this;

				Object.assign(vm.freqForm,{
					rat: '0',
					band: '1',
					freq: '',
				});
				Object.assign(vm.cellForm,{
					rat: '',
					band: '',
					earfcn: '',
					pci: ''
				});
				Object.assign(vm.bandForm,{
					rat: '',
					band: ''
				});
			},
			addFreqLock() {
				var vm = this;
				var freqLockList = vm.freqLockList;
				var nrList = [],lteList = [];
				freqLockList.map((item)=>{
					if(item.rat == '0'){
						lteList.push(item)
					}else{
						nrList.push(item)
					}
				})
				if((lteList.length == 2 && vm.freqForm.rat == '0') || (nrList.length == 10 && vm.freqForm.rat == '1')){
					vm.$message.warning('5G NR：No more than 10,4G LTE：No more than 2')
					return
				}
				vm.$refs.freqForm.validate(function(r){
					if(r) {
						vm.freqLockList.push(Object.assign({},vm.freqForm));
						vm.freq5gDlShow = false;
					}
				});
			},
			addCellLock() {
				var vm = this;

				vm.$refs.cellForm.validate(function(r){
					if(r) {
						vm.cellLockList.push(Object.assign({},vm.cellForm));
						vm.cell5gDlShow = false;
					}
				});
			},
			addBandLock() {
				var vm = this;
				
				vm.$refs.bandForm.validate(function(r){
					if(r) {
						vm.bandLockList.push(Object.assign({},vm.bandForm));
						vm.band5gDlShow = false;
					}
				});
			},
			deleteFreqLock(idx) {
				var vm = this;

				vm.freqLockList.splice(idx,1);
			},
			deleteCellLock(idx) {
				var vm = this;

				vm.cellLockList.splice(idx,1);
			},
			deleteBandLock(idx) {
				var vm = this;

				vm.bandLockList.splice(idx,1);
			},
			// cpe 设备型号改变事件
			cpeModelChange(){
				var vm = this;
				vm.queryForm.cpe_model = vm.cpePciForm.selectModel;
				vm.$refs.cpairgrid.clear();
			},
			// 5GCPE  scan Mode为freq lock时  rat改变事件
			freqLockRatChange(){
				var vm = this;
				vm.freqForm.band = '1';
				this.$refs.freqForm.validateField('freq');
			},
			// 5GCPE  scan Mode为freq lock时  band改变事件
			freqLockBandChange(){
				var vm = this;
				this.$refs.freqForm.validateField('freq');
			},
		},
		watch:{
			"cpePciForm.status":function(newVal){
				if(newVal == 'timing' && this.operType != 'view'){
					this.setTimeEnable = false
				}else{
					this.setTimeEnable = true
				}
				this.$refs.cpePciForm.validateField('exetime');
			},
			selection(val){
				var cpeCodes = val.map(function(item){
					return item.CPE_CODE;
				})
				this.cpePciForm.cpeCodes = cpeCodes.toString();
			},
			"cpePciForm.pciGroup":function(){
				var vm = this;
				var value = vm.cpePciForm.pciGroup.map(function(item){
					return item.value;
				})
				vm.cpePciForm.pciGroupStr = value.toString();
			},
			"cpePciForm.earfcnGroup":function(){
				var vm = this;
				var earfcn = vm.cpePciForm.earfcnGroup.map(function(item){
					return item.earfcn;
				})
				vm.cpePciForm.earfcnGroupStr = earfcn.toString();
			},
			"cpePciForm.pcionlylockGroup":function(){
				var vm = this;
				var pci = vm.cpePciForm.pcionlylockGroup.map(function(item){
					return item.pci
				})
				vm.cpePciForm.pcionlylockGroupStr = pci.toString();
			},
			freqLockList: {
				handler: function(rows) {
					var vm = this,
						list = [];

					(rows||[]).map(function(row){
						list.push(row.rat +','+ row.band +','+ row.freq);
					});

					vm.cpePciForm.nrFreq = list.join(';');
				},
				deep: true
			},
			cellLockList: {
				handler: function(rows) {
					var vm = this,
						list = [];

					(rows||[]).map(function(row){
						list.push(row.rat +','+ row.band +','+ row.earfcn +','+ row.pci);
					});

					vm.cpePciForm.nrPci = list.join(';');
				},
				deep: true
			},
			bandLockList: {
				handler: function(rows) {
					var vm = this,
						list = [];

					(rows||[]).map(function(row){
						list.push(row.rat +','+ row.band);
					});

					vm.cpePciForm.nrBand = list.join(';');
				},
				deep: true
			}
		},
		mounted(){
			eventBus.$off('add-cpe').$on('add-cpe',this.submit);
			eventBus.$off('get-cpe-info').$on('get-cpe-info',this.init);
			eventBus.$off('cancel-add-cpe').$on('cancel-add-cpe',this.cancelSubmit);
		}
	})
</script>