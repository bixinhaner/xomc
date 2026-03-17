<%@ page import="java.util.Locale" %>
<%@ page contentType="text/html;charset=UTF-8" %>
<%@ include file="/common/taglibs.jsp" %>

<style type="text/css">
#gnbBwpDetailsPage{
	height: 100%;
	width: 100%;
}
#gnbBwpDetailsPage .itemMainBoxCls{
	border-radius:10px;
	background:#fff;
	height:100%;
	width: 100%;
    display: flex;
    flex-direction: column;
    position: relative;
}
#gnbBwpDetailsPage .itemMainBoxTitle {
	height:36px;
	padding-left: 20px;
    line-height: 36px;
	font-size:14px;
	font-weight:bold;
	border-bottom: 1px solid #E9E9E9;
}
#gnbBwpDetailsPage .itemMainBoxCenter{
	width: calc(100% - 40px);
	flex:1;
	padding: 0px 20px;
	position: relative;
	overflow: auto;
}
#gnbBwpDetailsPage .itemMainBoxFooter{
    display: flex;
    align-items: center;
    border-top : 1px solid #E9E9E9;
	height:48px;
	background-color: #FFFFFF;
    box-sizing: border-box;
    width: 100%;
	padding-left: 20px;
}
#gnbBwpDetailsPage .rightContentCls .contentTableTitle{
	height: 40px;
	display: flex;
	align-items: center;
	justify-content: space-between;
	font-weight: 550;
	width: 100%;
}
#gnbBwpDetailsPage .rightContentCls .contentTableTitle>div:nth-child(1){
	font-size: 12px;
}
#gnbBwpDetailsPage .moreIpItemBoxCls{
	display: flex;
	flex-wrap: wrap;
	width: 100%;
}
#gnbBwpDetailsPage .leftAndRightItemCls{
	width:40%;
	min-width:400px;
	margin-bottom: 20px;
}
#gnbBwpDetailsPage .itemListBoxCls{
	padding-top: 5px;
}
#gnbBwpDetailsPage .itemCls{
	height: 24px;
	display: inline-block;
	line-height: 24px;
	border: 1px solid #4D84FF;
	box-sizing: border-box;
	padding: 0px 10px;
	margin-right: 10px;
	margin-bottom: 10px;
}
#gnbBwpDetailsPage .itemListBoxCls .el-icon-close{
	font-size: unset;
	position: unset;
	top: unset;
	right: unset;
}
#gnbBwpDetailsPage .leftAndRightItemCls .el-input__suffix{
	height: 26px;
	display: flex;
	align-items: center;
}
#gnbBwpDetailsPage .el-form-item{
	margin-bottom: 20px;
}
#gnbBwpDetailsPage .errorBoxCls{
	color:red;
	font-size:10px;
}
#gnbBwpDetailsPage .multiPlmnEnableBoxCls .el-form-item__label{
	padding-top: 13px;
	margin-right: 20px;
}
#gnbBwpDetailsPage .el-form-item .el-form-item__label{
	font-size: 12px;
}
</style>

<div id="gnbBwpDetailsPage">
	<div class="itemMainBoxCls">
		<div class="itemMainBoxTitle">
			<%=rb.getString("XiangQing")%>
			<span>(ID：{{rowDataInfo.DLBWP_idx}})</span>
			<!-- 按钮  关闭 -->
			<div class="headcloseBtn" style="top:6px;right:10px;" @click="closeBwpDetails">
				<span class="el-icon el-icon-close" style='position: unset; display: block; text-align: center;'></span>
			</div>
		</div>
		<div class="itemMainBoxCenter">
			<el-form :model='ruleForm' ref="ruleForm" :rules="rules" label-position="top">
				<div class="rightContentCls" >
					<!--PDCCH Common-->
					<div> 
						<div class="contentTableTitle">
							<div><span style="margin-right: 5px;" class="el-icon el-icon-operation-details"></span>PDCCH Common</div>
							<div></div>
						</div>
						<div style="display:flex;margin-left:16px;flex-wrap: wrap">
							<el-form-item prop='PC_CoresetZero' style="min-width:400px;" label="Coreset Zero" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.PC_CoresetZero'>
									<template slot="append"><%=rb.getString("FanWei")%>：0~15,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='PC_SearchSpaceZero' style="min-width:400px;" label="Search Space Zero" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.PC_SearchSpaceZero'>
									<template slot="append"><%=rb.getString("FanWei")%>：0~15,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='PC_SearchSpaceSIB1' style="min-width:400px;" label="Search Space SIB1" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.PC_SearchSpaceSIB1'>
									<template slot="append"><%=rb.getString("FanWei")%>：0~15,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='PC_SearchSpaceOtherSystemInformation' style="min-width:400px;" label="SearchSpaceOtherSystemInformation" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.PC_SearchSpaceOtherSystemInformation'>
									<template slot="append"><%=rb.getString("FanWei")%>：0~15,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='PC_PagingSearchSpace' style="min-width:400px;" label="Paging Search Space" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.PC_PagingSearchSpace'>
									<template slot="append"><%=rb.getString("FanWei")%>：0~15,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='PC_RaSearchSpace' style="min-width:400px;" label="Ra Search Space" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.PC_RaSearchSpace'>
									<template slot="append"><%=rb.getString("FanWei")%>：0~15,Integer</template>
								</el-input>
							</el-form-item>
						</div>
					</div>
				</div>
				<div class="rightContentCls" >
					<!--PDCCH Coreset Common-->
					<div> 
						<div class="contentTableTitle">
							<div><span style="margin-right: 5px;" class="el-icon el-icon-operation-details"></span>PDCCH Coreset Common</div>
						</div>
						<div style="display:flex;margin-left:16px;flex-wrap: wrap">
							<el-form-item prop='PCC_CoresetID' style="min-width:400px;" label="Coreset ID" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.PCC_CoresetID'>
									<template slot="append"><%=rb.getString("FanWei")%>：0~11,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='PCC_FeqDomainResources' style="min-width:400px;" label="Feq Domain Resources" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.PCC_FeqDomainResources'>
									<template slot="append">Length：45 Digit,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='PCC_NumSymbols' style="min-width:400px;" label="Num Symbols" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.PCC_NumSymbols'>
									<template slot="append"><%=rb.getString("FanWei")%>：1~3,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='PCC_CceRegMappingType' style="min-width:400px;" label="CceReg Mapping Type" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.PCC_CceRegMappingType'>
									<template slot="append"><%=rb.getString("FanWei")%>：0~1,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='PCC_CceRegBundleSize' style="min-width:400px;" label="CceReg Bundle Size" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.PCC_CceRegBundleSize'>
									<template slot="append"><%=rb.getString("FanWei")%>：0~2,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='PCC_CceInterleaverSize' style="min-width:400px;" label="Cce Interleaver Size" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.PCC_CceInterleaverSize'>
									<template slot="append"><%=rb.getString("FanWei")%>：0~2,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='PCC_CceShiftIndex' style="min-width:400px;" label="Cce Shift Index" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.PCC_CceShiftIndex'>
									<template slot="append"><%=rb.getString("FanWei")%>：0~274,Integer</template>
								</el-input>
							</el-form-item>
							<el-form-item prop='PCC_PrecoderGranularity' style="min-width:400px;" label="Precoder Granularity" label-width="160px" class='validate-item'>
								<el-input v-model.trim='ruleForm.PCC_PrecoderGranularity'>
									<template slot="append"><%=rb.getString("FanWei")%>：0~1,Integer</template>
								</el-input>
							</el-form-item>
						</div>
					</div>
				</div>
				<div class="rightContentCls" >
					<!--PDCCH Coreset Dedicated List-->
					<div> 
						<div class="contentTableTitle">
							<div><span style="margin-right: 5px;" class="el-icon el-icon-operation-details"></span>PDCCH Coreset Dedicated List</div>
							<div><span class="el-icon el-icon-circle-add" @click="addPCDDialogOpen('','add','PCD')"></span></div>
						</div>
						<div class="cellTableBoxCls" style="padding-bottom:20px;">
							<el-ctable
								ref="PCDTable" 
								:row-class-name="tableRowClassName"
								:rownumber="false" 
								id="PCDTable" 
								:data="ruleForm.PCDList" 
								height="200px"
								:pagination="false"
								style="border:1px solid #E9E9E9;"
							>
								<el-table-column label='<%=rb.getString("CaoZuo")%>' width="90">
									<template slot-scope="scope">
										<span class="el-icon el-icon-operation-edit" @click="addPCDDialogOpen(scope.row,'edit','PCD')" style="margin-right: 5px;"></span>
										<span class="el-icon el-icon-operation-delete" @click="delPCDList(scope.row,'PCD',event)" ></span>
									</template>
								</el-table-column>	
								<el-table-column label='ID' min-width="40" prop="PCD_idx" show-overflow-tooltip></el-table-column>
								<el-table-column label='Coreset ID' min-width="120" prop="PCD_CoresetID" show-overflow-tooltip></el-table-column>
								<el-table-column label='Feq Domain Resources' min-width="120" prop="PCD_FeqDomainResources" show-overflow-tooltip></el-table-column>
								<el-table-column label='CceReg Mapping Type' min-width="120" prop="PCD_CceRegMappingType" show-overflow-tooltip></el-table-column>
							</el-ctable>
							<el-form-item prop='PCDList' style="display:none;" label="" label-width="0px">
								<el-input v-model='ruleForm.PCDList'></el-input>
							</el-form-item>
						</div>
					</div>
				</div>
				<div class="rightContentCls" >
					<!--PDCCH Search Space Common List-->
					<div> 
						<div class="contentTableTitle">
							<div><span style="margin-right: 5px;" class="el-icon el-icon-operation-details"></span>PDCCH Search Space Common List</div>
							<div><span class="el-icon el-icon-circle-add" @click="addPSSCDialogOpen('','add','PSSC')"></span></div>
						</div>
						<div class="cellTableBoxCls" style="padding-bottom:20px;">
							<el-ctable
								ref="PSSCTable" 
								:row-class-name="tableRowClassName"
								:rownumber="false" 
								id="PSSCTable" 
								:data="ruleForm.PSSCList" 
								height="200px"
								:pagination="false"
								style="border:1px solid #E9E9E9;"
							>
								<el-table-column label='<%=rb.getString("CaoZuo")%>' width="90">
									<template slot-scope="scope">
										<span class="el-icon el-icon-operation-edit" @click="addPSSCDialogOpen(scope.row,'edit','PSSC')" style="margin-right: 5px;"></span>
										<span class="el-icon el-icon-operation-delete" @click="delPSSCList(scope.row,'PSSC',event)" ></span>
									</template>
								</el-table-column>	
								<el-table-column label='ID' min-width="40" prop="PSSC_idx" show-overflow-tooltip></el-table-column>
								<el-table-column label='Search Space ID' min-width="120" prop="PSSC_SearchSpaceID" show-overflow-tooltip></el-table-column>
								<el-table-column label='Coreset ID' min-width="120" prop="PSSC_CoresetID" show-overflow-tooltip></el-table-column>
								<el-table-column label='DciFormat00And10En' min-width="120" prop="PSSC_DciFormat00And10En" show-overflow-tooltip></el-table-column>
							</el-ctable>
							<el-form-item prop='PSSCList' style="display:none;" label="" label-width="0px">
								<el-input v-model='ruleForm.PSSCList'></el-input>
							</el-form-item>
						</div>
					</div>
				</div>
				<div class="rightContentCls" >
					<!--PDSCH Config-->
					<div> 
						<div class="contentTableTitle">
							<div><span style="margin-right: 5px;" class="el-icon el-icon-operation-details"></span>PDSCH Config</div>
						</div>
						<div style="display:flex;margin-left:16px;flex-wrap: wrap">
							<el-form-item prop='PDSCHC_DmrsAdditionalPosition' style="min-width:400px;" label="DmrsAdditionalPosition" label-width="160px">
								<el-select v-model='ruleForm.PDSCHC_DmrsAdditionalPosition'>
									<el-option label='pos0' value='0'></el-option>
									<el-option label='pos1' value='1'></el-option>
									<el-option label='pos3' value='2'></el-option>
									<el-option label='pos2' value='4095'></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='PDSCHC_MaxMimo' style="min-width:400px;" label="MaxMimo" label-width="160px">
								<el-select v-model='ruleForm.PDSCHC_MaxMimo'>
									<el-option label='0' value='0'></el-option>
									<el-option label='1' value='1'></el-option>
									<el-option label='2' value='2'></el-option>
									<el-option label='3' value='3'></el-option>
									<el-option label='4' value='4'></el-option>
									<el-option label='5' value='5'></el-option>
									<el-option label='6' value='6'></el-option>
									<el-option label='7' value='7'></el-option>
									<el-option label='8' value='8'></el-option>
								</el-select>
							</el-form-item>
							<el-form-item prop='PDSCHC_McsTable' style="min-width:400px;" label="McsTable" label-width="160px">
								<el-select v-model='ruleForm.PDSCHC_McsTable'>
									<el-option label='0' value='0'></el-option>
									<el-option label='1' value='1'></el-option>
									<el-option label='2' value='2'></el-option>
								</el-select>
							</el-form-item>
						</div>
					</div>
				</div>
				<div class="rightContentCls" >
					<!--PDSCH Search Space Common List-->
					<div> 
						<div class="contentTableTitle">
							<div><span style="margin-right: 5px;" class="el-icon el-icon-operation-details"></span>PDSCH Search Space Common List</div>
							<div><span class="el-icon el-icon-circle-add" @click="addPDSCHSSCDialogOpen('','add','PDSCHSSC')"></span></div>
						</div>
						<div class="cellTableBoxCls" style="padding-bottom:20px;">
							<el-ctable
								ref="PDSCHSSCTable" 
								:row-class-name="tableRowClassName"
								:rownumber="false" 
								id="PDSCHSSCTable" 
								:data="ruleForm.PDSCHSSCList" 
								height="200px"
								:pagination="false"
								style="border:1px solid #E9E9E9;"
							>
								<el-table-column label='<%=rb.getString("CaoZuo")%>' width="90">
									<template slot-scope="scope">
										<span class="el-icon el-icon-operation-edit" @click="addPDSCHSSCDialogOpen(scope.row,'edit','PDSCHSSC')" style="margin-right: 5px;"></span>
										<span class="el-icon el-icon-operation-delete" @click="delPDSCHSSCList(scope.row,'PDSCHSSC',event)" ></span>
									</template>
								</el-table-column>	
								<el-table-column label='ID' min-width="40" prop="PDSCHSSC_idx" show-overflow-tooltip></el-table-column>
								<el-table-column label='Start Symbol' min-width="120" prop="PDSCHSSC_StartSymbol" show-overflow-tooltip></el-table-column>
								<el-table-column label='Length' min-width="120" prop="PDSCHSSC_Length" show-overflow-tooltip></el-table-column>
								<el-table-column label='Start Symbol And Length' min-width="200" prop="PDSCHSSC_StartSymbolAndLength" show-overflow-tooltip></el-table-column>
							</el-ctable>
							<el-form-item prop='PDSCHSSCList' style="display:none;" label="" label-width="0px">
								<el-input v-model='ruleForm.PDSCHSSCList'></el-input>
							</el-form-item>
						</div>
					</div>
				</div>
				<div class="rightContentCls" >
					<!--PDSCH Dedicated List-->
					<div> 
						<div class="contentTableTitle">
							<div><span style="margin-right: 5px;" class="el-icon el-icon-operation-details"></span>PDSCH Dedicated List</div>
							<div><span class="el-icon el-icon-circle-add" @click="addPDSCHDDialogOpen('','add','PDSCHD')"></span></div>
						</div>
						<div class="cellTableBoxCls" style="padding-bottom:20px;">
							<el-ctable
								ref="PDSCHDTable" 
								:row-class-name="tableRowClassName"
								:rownumber="false" 
								id="PDSCHDTable" 
								:data="ruleForm.PDSCHDList" 
								height="200px"
								:pagination="false"
								style="border:1px solid #E9E9E9;"
							>
								<el-table-column label='<%=rb.getString("CaoZuo")%>' width="90">
									<template slot-scope="scope">
										<span class="el-icon el-icon-operation-edit" @click="addPDSCHDDialogOpen(scope.row,'edit','PDSCHD')" style="margin-right: 5px;"></span>
										<span class="el-icon el-icon-operation-delete" @click="delPDSCHDList(scope.row,'PDSCHD',event)" ></span>
									</template>
								</el-table-column>	
								<el-table-column label='ID' min-width="40" prop="PDSCHD_idx" show-overflow-tooltip></el-table-column>
								<el-table-column label='Start Symbol' min-width="120" prop="PDSCHD_StartSymbol" show-overflow-tooltip></el-table-column>
								<el-table-column label='Length' min-width="120" prop="PDSCHD_Length" show-overflow-tooltip></el-table-column>
								<el-table-column label='Start Symbol And Length' min-width="200" prop="PDSCHD_StartSymbolAndLength" show-overflow-tooltip></el-table-column>
							</el-ctable>
							<el-form-item prop='PDSCHDList' style="display:none;" label="" label-width="0px">
								<el-input v-model='ruleForm.PDSCHDList'></el-input>
							</el-form-item>
						</div>
					</div>
				</div>
			</el-form>
		</div>
		<div class='itemMainBoxFooter'>
			<el-button type="primary" @click="settingsSubmit"><%=rb.getString("QueDing")%></el-button>
			<el-button @click="closeSettings" ><%=rb.getString("QuXiao")%></el-button>
		</div>
	</div>
	<!-- PCD 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addPCDDialogShow" @close="closeAddPCDDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<el-form ref="addPCDDialogForm" :model='addPCDDialogForm' :rules='addPCDDialogRules' label-position="top">     		     			            
			<el-form-item prop='PCD_CoresetID' style="min-width:400px;" label="Coreset ID" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPCDDialogForm.PCD_CoresetID'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~11,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PCD_FeqDomainResources' style="min-width:400px;" label="Feq Domain Resources" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPCDDialogForm.PCD_FeqDomainResources'>
                    <template slot="append">Length：45 Digit,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PCD_NumSymbols' style="min-width:400px;" label="Num Symbols" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPCDDialogForm.PCD_NumSymbols'>
                    <template slot="append"><%=rb.getString("FanWei")%>：1~3,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PCD_CceRegMappingType' style="min-width:400px;" label="CceReg Mapping Type" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPCDDialogForm.PCD_CceRegMappingType'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~1,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PCD_CceRegBundleSize' style="min-width:400px;" label="CceReg Bundle Size" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPCDDialogForm.PCD_CceRegBundleSize'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~2,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PCD_CceInterleaverSize' style="min-width:400px;" label="Cce Interleaver Size" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPCDDialogForm.PCD_CceInterleaverSize'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~2,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PCD_CceShiftIndex' style="min-width:400px;" label="Cce Shift Index" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPCDDialogForm.PCD_CceShiftIndex'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~274,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PCD_PrecoderGranularity' style="min-width:400px;" label="Precoder Granularity" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPCDDialogForm.PCD_PrecoderGranularity'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~1,Integer</template>
                </el-input>
            </el-form-item>
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addPCDDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addPCDDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- PSSC 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addPSSCDialogShow" @close="closeAddPSSCDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<el-form ref="addPSSCDialogForm" :model='addPSSCDialogForm' :rules='addPSSCDialogRules' label-position="top">     		     			            
			<el-form-item prop='PSSC_SearchSpaceID' style="min-width:400px;" label="Search Space ID" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPSSCDialogForm.PSSC_SearchSpaceID'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~39,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PSSC_CoresetID' style="min-width:400px;" label="Coreset ID" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPSSCDialogForm.PSSC_CoresetID'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~11,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PSSC_PdcchSlotPeriodicity' style="min-width:400px;" label="PdcchSlotPeriodicity" label-width="160px">
				<el-select v-model='addPSSCDialogForm.PSSC_PdcchSlotPeriodicity'>
					<el-option v-for="item in PdcchSlotPeriodicityList" :label='item' :value='item'></el-option>
				</el-select>
			</el-form-item>
			<el-form-item prop='PSSC_PdcchSlotOffset' style="min-width:400px;" label="PdcchSlotOffset" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPSSCDialogForm.PSSC_PdcchSlotOffset'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~2559,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PSSC_SearchSpaceDuration' style="min-width:400px;" label="SearchSpaceDuration" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPSSCDialogForm.PSSC_SearchSpaceDuration'>
                    <template slot="append"><%=rb.getString("FanWei")%>：2~2559,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PSSC_PdcchSymbolsInSlot' style="min-width:400px;" label="PdcchSymbolsInSlot" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPSSCDialogForm.PSSC_PdcchSymbolsInSlot'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~16383,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PSSC_PdcchCandidatesAggLevel1' style="min-width:400px;" label="PdcchCandidatesAggLevel1" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPSSCDialogForm.PSSC_PdcchCandidatesAggLevel1'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~7,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PSSC_PdcchCandidatesAggLevel2' style="min-width:400px;" label="PdcchCandidatesAggLevel2" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPSSCDialogForm.PSSC_PdcchCandidatesAggLevel2'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~7,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PSSC_PdcchCandidatesAggLevel4' style="min-width:400px;" label="PdcchCandidatesAggLevel4" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPSSCDialogForm.PSSC_PdcchCandidatesAggLevel4'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~7,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PSSC_PdcchCandidatesAggLevel8' style="min-width:400px;" label="PdcchCandidatesAggLevel8" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPSSCDialogForm.PSSC_PdcchCandidatesAggLevel8'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~7,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PSSC_PdcchCandidatesAggLevel16' style="min-width:400px;" label="PdcchCandidatesAggLevel16" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPSSCDialogForm.PSSC_PdcchCandidatesAggLevel16'>
                    <template slot="append"><%=rb.getString("FanWei")%>：0~7,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PSSC_SearchSpaceType' style="min-width:400px;" label="Search Space Type" label-width="160px">
				<el-select v-model='addPSSCDialogForm.PSSC_SearchSpaceType'>
					<el-option label='Common' value='0'></el-option>
					<el-option label='Dedicated' value='1'></el-option>
				</el-select>
			</el-form-item>
			<el-form-item prop='PSSC_DciFormat00And10En' style="min-width:400px;" label="DciFormat00And10En" label-width="160px">
				<el-select v-model='addPSSCDialogForm.PSSC_DciFormat00And10En'>
					<el-option label='DCI00_10' value='0'></el-option>
					<el-option label='DCI01_11' value='1'></el-option>
				</el-select>
			</el-form-item>
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addPSSCDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addPSSCDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- PDSCHSSC 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addPDSCHSSCDialogShow" @close="closeAddPDSCHSSCDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<el-form ref="addPDSCHSSCDialogForm" :model='addPDSCHSSCDialogForm' :rules='addPDSCHSSCDialogRules' label-position="top">     		     			            
			<el-form-item prop='PDSCHSSC_StartSymbol' style="min-width:400px;" label="Start Symbol" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPDSCHSSCDialogForm.PDSCHSSC_StartSymbol' @change="PDSCHSSC_StartSymbolAndLengthChange">
                    <template slot="append"><%=rb.getString("FanWei")%>：0~13,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PDSCHSSC_Length' style="min-width:400px;" label="Length" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPDSCHSSCDialogForm.PDSCHSSC_Length' @change="PDSCHSSC_StartSymbolAndLengthChange">
                    <template slot="append"><%=rb.getString("FanWei")%>：1~14,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PDSCHSSC_StartSymbolAndLength' style="min-width:400px;" label="Start Symbol And Length" label-width="160px">
                <el-input v-model.trim='addPDSCHSSCDialogForm.PDSCHSSC_StartSymbolAndLength' :disabled="true"></el-input>
            </el-form-item>
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addPDSCHSSCDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addPDSCHSSCDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
	<!-- PDSCHD 新增 弹窗 -->
	<el-dialog class="gnbConfigAddDialog" top="25vh" :title="gnbConfigAddDialogTitle" width="50%" :visible.sync="addPDSCHDDialogShow" @close="closeAddPDSCHDDialog" :close-on-click-modal="false" :modal-append-to-body="false">		
		<el-form ref="addPDSCHDDialogForm" :model='addPDSCHDDialogForm' :rules='addPDSCHDDialogRules' label-position="top">     		     			            
			<el-form-item prop='PDSCHD_StartSymbol' style="min-width:400px;" label="Start Symbol" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPDSCHDDialogForm.PDSCHD_StartSymbol' @change="PDSCHD_StartSymbolAndLengthChange">
                    <template slot="append"><%=rb.getString("FanWei")%>：0~13,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PDSCHD_Length' style="min-width:400px;" label="Length" label-width="160px" class='validate-item'>
                <el-input v-model.trim='addPDSCHDDialogForm.PDSCHD_Length' @change="PDSCHD_StartSymbolAndLengthChange">
                    <template slot="append"><%=rb.getString("FanWei")%>：1~14,Integer</template>
                </el-input>
            </el-form-item>
			<el-form-item prop='PDSCHD_StartSymbolAndLength' style="min-width:400px;" label="Start Symbol And Length" label-width="160px">
                <el-input v-model.trim='addPDSCHDDialogForm.PDSCHD_StartSymbolAndLength' :disabled="true"></el-input>
            </el-form-item>
        </el-form> 
       	<div slot="footer" class="importFooter">
          	<el-button type="primary" @click="addPDSCHDDialogSubmit"><%=rb.getString("QueDing")%></el-button>
            <el-button @click="addPDSCHDDialogShow = false"><%=rb.getString("QuXiao")%></el-button>
         </div>			
	</el-dialog>
</div>

<script>
var regIp = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/,
	regKey = /^[A-Fa-f0-9]{32}$/,
	regNumber = /^[0-9]{15}$/;
var gnbBwpDetailsPage = new Vue({
	el: '#gnbBwpDetailsPage', 
	data() {
		var vm = this,
			validateRange = (rule,value,callback)=>{
				var min = rule.min;
				var max = rule.max;
				var mag = rule.mag;
				var isRequired = rule.isRequired;
				var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;

				if(value == '' || value == undefined || value == null){
					if(isRequired){
						callback(new Error(mag))
					}else{
						callback();
					}
				}else{
					if(reg.test(value) && value >= min && value <= max){
						callback();
					}else{
						callback(new Error(mag))
					}
				}
			},
			validateFeqDomainResources = (rule,value,callback) => {
				var reg = /^[0-9]{45}$/
				if(value == '' || value == undefined || value == null){
					callback(new Error('Length<%=rb.getString("MaoHao")%> 45 Digit <%=rb.getString("ZhengXing")%>'))
				}else{
					if(reg.test(value)){
						callback();
					}else{
						callback(new Error('Length<%=rb.getString("MaoHao")%> 45 Digit <%=rb.getString("ZhengXing")%>'))
					}
				}
			};
		return {
			rowDataInfo: [],
			smallCellCode:'',
			ruleForm:{
				PCDList:[],
				PSSCList:[],
				PDSCHSSCList:[],
				PDSCHDList:[],

				PC_CoresetZero:'',
				PC_SearchSpaceZero:'',
				PC_SearchSpaceSIB1:'',
				PC_SearchSpaceOtherSystemInformation:'0',
				PC_PagingSearchSpace:'',
				PC_RaSearchSpace:'',

				PCC_CoresetID:'',
				PCC_FeqDomainResources:'',
				PCC_NumSymbols:'',
				PCC_CceRegMappingType:'',
				PCC_CceRegBundleSize:'',
				PCC_CceInterleaverSize:'',
				PCC_CceShiftIndex:'',
				PCC_PrecoderGranularity:'',

				PDSCHC_DmrsAdditionalPosition:'0',
				PDSCHC_MaxMimo:'2',
				PDSCHC_McsTable:'0',
			},
			rules:{
				PC_CoresetZero:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:15,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~15,Integer'}
				],
				PC_SearchSpaceZero:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:15,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~15,Integer'}
				],
				PC_SearchSpaceSIB1:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:15,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~15,Integer'}
				],
				PC_SearchSpaceOtherSystemInformation:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:15,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~15,Integer'}
				],
				PC_PagingSearchSpace:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:15,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~15,Integer'}
				],
				PC_RaSearchSpace:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:15,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~15,Integer'}
				],
				PCC_CoresetID:[
					{validator:validateRange,min:0,max:11,isRequired:false,mag:'<%=rb.getString("FanWei")%>0~11,Integer'}
				],
				PCC_FeqDomainResources:[
					{validator:validateFeqDomainResources}
				],
				PCC_NumSymbols:[
					{validator:validateRange,min:1,max:3,isRequired:false,mag:'<%=rb.getString("FanWei")%>1~3,Integer'}
				],
				PCC_CceRegMappingType:[
					{validator:validateRange,min:0,max:1,isRequired:false,mag:'<%=rb.getString("FanWei")%>0~1,Integer'}
				],
				PCC_CceRegBundleSize:[
					{validator:validateRange,min:0,max:2,isRequired:false,mag:'<%=rb.getString("FanWei")%>0~2,Integer'}
				],
				PCC_CceInterleaverSize:[
					{validator:validateRange,min:0,max:2,isRequired:false,mag:'<%=rb.getString("FanWei")%>0~2,Integer'}
				],
				PCC_CceShiftIndex:[
					{validator:validateRange,min:0,max:274,isRequired:false,mag:'<%=rb.getString("FanWei")%>0~274,Integer'}
				],
				PCC_PrecoderGranularity:[
					{validator:validateRange,min:0,max:1,isRequired:false,mag:'<%=rb.getString("FanWei")%>0~1,Integer'}
				],
			},
			casts:{
				'B03E77F813ABFE7FB21855D4213678D3':'PC_CoresetZero',
				'B3E25AE53CF80DDF9B5AC8DA54F9D921':'PC_SearchSpaceZero',
				'A65CE8D56AD78CC7B7F92AACBA4AC3E0':'PC_SearchSpaceSIB1',
				'10A1CF9B17ACA6B5D8153A9383B2EB29':'PC_SearchSpaceOtherSystemInformation',
				'C6391E07216561C141EF946A4C0B285A':'PC_PagingSearchSpace',
				'3E693C4AED172F61D630176EB5514262':'PC_RaSearchSpace',

				'7CB19E81C355223B32A6BA7B9DE56EF4':'PCC_CoresetID',
				'9A3438EB70B57D9F04D92BFF735FB1BE':'PCC_FeqDomainResources',
				'86C2463E87245E3B62EC34FC295908A0':'PCC_NumSymbols',
				'6581C91FFCB40810F6AB6FC67EB7ECDF':'PCC_CceRegMappingType',
				'03B7F96860BFB3798C03662F9BD25057':'PCC_CceRegBundleSize',
				'B3FD8B3B192BDACD736FD789A83CD1B6':'PCC_CceInterleaverSize',
				'C79BBF158050C60F73194326BBA5FBF2':'PCC_CceShiftIndex',
				'9626F2D880485FD7754B759E3ECD4352':'PCC_PrecoderGranularity',

				'6575EA789C7FCF689B2DD6D6E9E186AE':'PCDList',
				'BB3B6882FC51B07BFE0F9CF7792D455A':'PCD_idx',
				'2CDE81516DAD023E8647060B88A21093':'PCD_CoresetID',
				'1732E609F87D14D007107D24EB4D09B9':'PCD_FeqDomainResources',
				'DA8FDF266453443D45F99B0B5FEF2C07':'PCD_NumSymbols',
				'3B4EC99AEDC5F91FB25E73B931F9F1D8':'PCD_CceRegMappingType',
				'3569D89C87FEDC99BE3E215EFC11C9BF':'PCD_CceRegBundleSize',
				'96988D5E3412ECAF1722387F904832EC':'PCD_CceInterleaverSize',
				'C6054DD6B2EF806020F999EA2321F30F':'PCD_CceShiftIndex',
				'E827FF7170E97CBF731B632F40804749':'PCD_PrecoderGranularity',

				'78566810F703C7C562E584E06CD56E61':'PSSCList',
				'CFD9AA6782FFA7D3AB2B0489367927B0':'PSSC_idx',
				'5535B3361C2242384C8CB3C8924E8240':'PSSC_SearchSpaceID',
				'5FFDDC889481A02359C41E3AB4FE0783':'PSSC_CoresetID',
				'C7C6EFC072DF092091CC7CFF7E730837':'PSSC_PdcchSlotPeriodicity',
				'C007391BEAB70155CF2B7312E76DC0D6':'PSSC_PdcchSlotOffset',
				'8C7E7616EE2ADE9CAF4BE25F2CF63AF9':'PSSC_SearchSpaceDuration',
				'25DDD99F18A085DD064A3EB9FF785838':'PSSC_PdcchSymbolsInSlot',
				'D34EAD2E361E39A618514FB1A3EEF53D':'PSSC_PdcchCandidatesAggLevel1',
				'0FCD14C32279F3D45D0710453788D13C':'PSSC_PdcchCandidatesAggLevel2',
				'E1059E0960A29E44ED378CB0732C4DA9':'PSSC_PdcchCandidatesAggLevel4',
				'1C693D267F7AB25E76B9D7E186AB4D82':'PSSC_PdcchCandidatesAggLevel8',
				'3E14EA8ABF208B27F19DAC2F39D5D282':'PSSC_PdcchCandidatesAggLevel16',
				'45B78476F29BD5754B02E8076638AAA6':'PSSC_SearchSpaceType',
				'B154A38869F47412665989C5D4F1FF36':'PSSC_DciFormat00And10En',

				'91A4FD63FF109B64FABCEAF46863A94B':'PDSCHC_DmrsAdditionalPosition',
				'EC384CE6F251888001C5F2C2D07D65E0':'PDSCHC_MaxMimo',
				'478F544D27BAF2B92A8687A478A9D8F3':'PDSCHC_McsTable',

				'5AE506BD39C41973416894D7280B8F4A':'PDSCHSSCList',
				'AA3AA8A643272520C1F96F1CD9AE502B':'PDSCHSSC_idx',
				'B9952A5F420C38D3B43EB280D664ACC9':'PDSCHSSC_StartSymbolAndLength',

				'27418176D4F9B87E80CE3FA012579841':'PDSCHDList',
				'BA1A172362789DAB986C4AC7F5AED02A':'PDSCHD_idx',
				'3775A5CA31ACEB996D605EF13C4CA2F4':'PDSCHD_StartSymbolAndLength',
			},
			codeList:[],
			optType:'',
			tbType:'',
			addPCDDialogShow:false,
			addPCDDialogForm:{
				PCD_CoresetID:'',
				PCD_FeqDomainResources:'',
				PCD_NumSymbols:'',
				PCD_CceRegMappingType:'',
				PCD_CceRegBundleSize:'',
				PCD_CceInterleaverSize:'',
				PCD_CceShiftIndex:'',
				PCD_PrecoderGranularity:'',
			},
			addPCDDialogRules:{
				PCD_CoresetID:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:11,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~11,Integer'}
				],
				PCD_FeqDomainResources:[
					{required:true,trigger:'blur'},
					{validator:validateFeqDomainResources}
				],
				PCD_NumSymbols:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:1,max:3,isRequired:true,mag:'<%=rb.getString("FanWei")%>1~3,Integer'}
				],
				PCD_CceRegMappingType:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:1,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~1,Integer'}
				],
				PCD_CceRegBundleSize:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:2,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~2,Integer'}
				],
				PCD_CceInterleaverSize:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:2,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~2,Integer'}
				],
				PCD_CceShiftIndex:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:274,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~274,Integer'}
				],
				PCD_PrecoderGranularity:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:1,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~1,Integer'}
				],
			},
			addPSSCDialogShow:false,
			addPSSCDialogForm:{
				PSSC_SearchSpaceID:'',
				PSSC_CoresetID:'',
				PSSC_PdcchSlotPeriodicity:'1',
				PSSC_PdcchSlotOffset:'',
				PSSC_SearchSpaceDuration:'',
				PSSC_PdcchSymbolsInSlot:'',
				PSSC_PdcchCandidatesAggLevel1:'',
				PSSC_PdcchCandidatesAggLevel2:'',
				PSSC_PdcchCandidatesAggLevel4:'',
				PSSC_PdcchCandidatesAggLevel8:'',
				PSSC_PdcchCandidatesAggLevel16:'',
				PSSC_SearchSpaceType:'0',
				PSSC_DciFormat00And10En:'1',
			},
			addPSSCDialogRules:{
				PSSC_SearchSpaceID:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:39,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~39,Integer'}
				],
				PSSC_CoresetID:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:11,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~11,Integer'}
				],
				PSSC_PdcchSlotOffset:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:2559,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~2559,Integer'}
				],
				PSSC_SearchSpaceDuration:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:2,max:2559,isRequired:true,mag:'<%=rb.getString("FanWei")%>2~2559,Integer'}
				],
				PSSC_PdcchSymbolsInSlot:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:16383,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~16383,Integer'}
				],
				PSSC_PdcchCandidatesAggLevel1:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:7,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~7,Integer'}
				],
				PSSC_PdcchCandidatesAggLevel2:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:7,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~7,Integer'}
				],
				PSSC_PdcchCandidatesAggLevel4:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:7,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~7,Integer'}
				],
				PSSC_PdcchCandidatesAggLevel8:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:7,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~7,Integer'}
				],
				PSSC_PdcchCandidatesAggLevel16:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:7,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~7,Integer'}
				],
			},
			addPDSCHSSCDialogShow:false,
			addPDSCHSSCDialogForm:{
				PDSCHSSC_StartSymbol:'',
				PDSCHSSC_Length:'',
				PDSCHSSC_StartSymbolAndLength:'',
			},
			addPDSCHSSCDialogRules:{
				PDSCHSSC_StartSymbol:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:13,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~13,Integer'}
				],
				PDSCHSSC_Length:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:1,max:14,isRequired:true,mag:'<%=rb.getString("FanWei")%>1~14,Integer'}
				],
			},
			addPDSCHDDialogShow:false,
			addPDSCHDDialogForm:{
				PDSCHD_StartSymbol:'',
				PDSCHD_Length:'',
				PDSCHD_StartSymbolAndLength:'',
			},
			addPDSCHDDialogRules:{
				PDSCHD_StartSymbol:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:0,max:13,isRequired:true,mag:'<%=rb.getString("FanWei")%>0~13,Integer'}
				],
				PDSCHD_Length:[
					{required:true,trigger:'blur'},
					{validator:validateRange,min:1,max:14,isRequired:true,mag:'<%=rb.getString("FanWei")%>1~14,Integer'}
				],
			},
		};
	},
	computed: {
		gnbConfigAddDialogTitle(){
			return this.optType == 'add' ? '<%=rb.getString("TianJia")%>' : '<%=rb.getString("XiuGai")%>'
		},
		PdcchSlotPeriodicityList(){
			var list = ['1','2','4','5','10','15','16','20','40','80','160','320','640','1280','2560'],
				fotList = [];
			// list.map((item,index)=>{
			// 	var obj = {};
			// 	obj.label = item;
			// 	obj.value = index+'';
			// 	fotList.push(obj);
			// });
			// return fotList
			return list
		},
	},
	methods: {
		init(row,code){
			var vm = this;
			vm.rowDataInfo = row;
			vm.smallCellCode = code;
			var codeList=[];
			Object.keys(vm.casts).forEach(function(key){
				codeList.push(key)
			});
			vm.codeList = codeList;
			vm.getParamData(vm.smallCellCode,'23322');
		},
		getParamData(code,id) {
			var vm = this,
				codes = [],
				url = '${ctx}/cell/quicksettings/getParamNodeTreeAndData.action',
				params = {
					id: id,
					bwpIndex:vm.rowDataInfo.DLBWP_idx,
					cellIndex:'1',
					smallCellCode: code
				};
			axios.post(url, stringify(params)).then(function(res){
				var data = res.data;
				vm.resetFormData();
				if(data && Array.isArray(data)) {
					data.map(function(item){
						item.groups.map(function(group){
							group.list.map(function(m){
								if(m.type == 'list'){
									vm.initTable(m.url,m.label);
								}else{
									codes.push(m.name);
									// 执行赋值
									vm.setValue(m);
								}
							});
						});
					});
					initForm(vm.$refs.ruleForm);
				}
			});
		},
		// 映射赋值
		setValue(item) {
			var vm = this,
			code = item.name,
			value = item.value;

			// indexs是否含有
			var key = vm.casts[code];
			try{
				if(key){
					vm.ruleForm[key] = value;
				}
			}catch(e){}
		},
		initTable(url,type){
			var vm = this,codes = [];
			var params = {
					smallCellCode : vm.smallCellCode,
					bwpIndex:vm.rowDataInfo.DLBWP_idx,
			}
			axios.post(url,stringify(params)).then(res=>{
				var data = res.data;
				if(data.rows){
					data.rows.map(item=>{
						var obj = {};
						for(var key in item){
							codes.push(key);
							obj[vm.casts[key]] = item[key]
						}
						if(type == 'PDCCH Coreset Dedicated List'){
							vm.ruleForm.PCDList.push(obj);
						}else if(type == 'PDCCH Search Space Common List'){
							vm.ruleForm.PSSCList.push(obj);
						}else if(type == 'PDSCH Search Space Common List'){
							Object.keys(obj).map((paramKey)=>{
								if(paramKey == 'PDSCHSSC_StartSymbolAndLength'){
									if(obj[paramKey]){
										var quotientNum = parseInt(obj[paramKey]/14),
											remainder = parseInt(obj[paramKey]%14);
										if(quotientNum <= 7){
											var StartSymbol = remainder,
												Length = quotientNum + 1;
											if(Length <= 14 - StartSymbol){
												obj.PDSCHSSC_StartSymbol = remainder;
												obj.PDSCHSSC_Length = quotientNum + 1;
											}else{
												obj.PDSCHSSC_StartSymbol = 13 - remainder;
												obj.PDSCHSSC_Length = 15 - quotientNum;
											}
										}else{
											obj.PDSCHSSC_StartSymbol = 13 - remainder;
											obj.PDSCHSSC_Length = 15 - quotientNum;
										}

									}else{
										obj.PDSCHSSC_StartSymbol = '';
										obj.PDSCHSSC_Length = '';
									}
								}
							})
							vm.ruleForm.PDSCHSSCList.push(obj);
						}else if(type == 'PDSCH Dedicated List'){
							Object.keys(obj).map((paramKey)=>{
								if(paramKey == 'PDSCHD_StartSymbolAndLength' ){
									if(obj[paramKey]){
										var quotientNum = parseInt(obj[paramKey]/14),
											remainder = parseInt(obj[paramKey]%14);
										if(quotientNum <= 7){
											var StartSymbol = remainder,
												Length = quotientNum + 1;
											if(Length <= 14 - StartSymbol){
												obj.PDSCHD_StartSymbol = remainder;
												obj.PDSCHD_Length = quotientNum + 1;
											}else{
												obj.PDSCHD_StartSymbol = 13 - remainder;
												obj.PDSCHD_Length = 15 - quotientNum;
											}
										}else{
											obj.PDSCHD_StartSymbol = 13 - remainder;
											obj.PDSCHD_Length = 15 - quotientNum;
										}
									}else{
										obj.PDSCHD_StartSymbol = '';
										obj.PDSCHD_Length = '';
									}
								}
							})
							vm.ruleForm.PDSCHDList.push(obj);
						}
					})
					initForm(vm.$refs.ruleForm);
				}
			})
		},
		// 重置form数据
		resetFormData(){
			var vm =this;
				params={
					PCDList:[],
					PSSCList:[],
					PDSCHSSCList:[],
					PDSCHDList:[],

					PC_CoresetZero:'',
					PC_SearchSpaceZero:'',
					PC_SearchSpaceSIB1:'',
					PC_SearchSpaceOtherSystemInformation:'0',
					PC_PagingSearchSpace:'',
					PC_RaSearchSpace:'',

					PCC_CoresetID:'',
					PCC_FeqDomainResources:'',
					PCC_NumSymbols:'',
					PCC_CceRegMappingType:'',
					PCC_CceRegBundleSize:'',
					PCC_CceInterleaverSize:'',
					PCC_CceShiftIndex:'',
					PCC_PrecoderGranularity:'',

					PDSCHC_DmrsAdditionalPosition:'0',
					PDSCHC_MaxMimo:'2',
					PDSCHC_McsTable:'0',
				};
			Object.assign(vm.ruleForm,params);
		},
		// 判断是否为空
		isNull(val){
			if(val==undefined || val == null || val =="") return true;
			else return false;
		},
		// 验证输入的是否是整数
		isInteger(str) {
			if(str.length==0){
				return false;
			}
			var reg = /^(0|[1-9][0-9]*|-[1-9][0-9]*)$/;
			if(!reg.test(str)){
				return false;
			}
			return true;  
		},
		tableRowClassName({row,rowIndex}){
			if(row.operateType && row.operateType == 'remove'){
				return 'hidden-row'
			}
			return ''
		},
		// 打开新增 PCD 弹窗
		addPCDDialogOpen(row,optType,tbType){
			var vm = this;
			vm.tbType = tbType;
			vm.optType = optType;
			if(vm.optType == 'edit'){
				Object.assign(vm.addPCDDialogForm,row)
			}
			vm.addPCDDialogShow = true;
		},
		// 新增 PCD 提交
		addPCDDialogSubmit(){
			var vm = this,
				params = {},
				codes = {
					'PCD':'PCDList',
				},
				idxStr = vm.tbType + '_idx',
				optTb = codes[vm.tbType];
			Object.keys(vm.addPCDDialogForm).forEach(function(key){
				params[key] = vm.addPCDDialogForm[key]
			})
			if(vm.addPCDDialogForm.operateType){
				params.operateType = vm.addPCDDialogForm.operateType
			}
			vm.$refs.addPCDDialogForm.validate(function(valid){
				if(valid){
					if(vm.optType == 'add'){
						params.operateType = 'add'
						if(vm.ruleForm[optTb].length == 0){
							params[idxStr] = '1'
						}else{
							var idList=[];
							vm.ruleForm[optTb].map((item)=>{
								idList.push(item[idxStr]);
							})
							params[idxStr] = vm.createId(1,idList); 
						}
						vm.ruleForm[optTb].push(params);
					}else{
						if(params.operateType && params.operateType == 'add'){
							params.operateType = 'add'
						}else{
							params.operateType = 'edit';
						}
						var idx='';
						vm.ruleForm[optTb].map((item,index)=>{
							if(item[idxStr] == params[idxStr]){
								idx = index
							}
						})
						Object.assign(vm.ruleForm[optTb][idx],params)
					}
					vm.addPCDDialogShow = false;
				}
			})
		},
		// 删除 PCD
		delPCDList(row,tbType){
			var vm = this,
				codes = {
					'PCD':'PCDList',
				},
				idxStr = tbType + '_idx',
				optTb = codes[tbType];
			var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
			vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(()=>{
				var delFlag=false;
				vm.ruleForm[optTb].map(function(item,index){
					if(item[idxStr] == row[idxStr]){
						if(item.operateType == 'add'){
							delFlag = true;
						}else{
							var params = item;
							params.operateType = 'remove';
							vm.$set(vm.ruleForm[optTb],index,params);
						}
					}
				})
				if(delFlag){
					vm.ruleForm[optTb] = vm.ruleForm[optTb].filter((items)=>{
						return items[idxStr] != row[idxStr]
					})
				}
			})
		},
		// 关闭 PCD 弹窗
		closeAddPCDDialog(){
			var vm = this,
				params = {
					PCD_CoresetID:'',
					PCD_FeqDomainResources:'',
					PCD_NumSymbols:'',
					PCD_CceRegMappingType:'',
					PCD_CceRegBundleSize:'',
					PCD_CceInterleaverSize:'',
					PCD_CceShiftIndex:'',
					PCD_PrecoderGranularity:'',
				};
			Object.assign(vm.addPCDDialogForm,params);
			vm.$refs.addPCDDialogForm.clearValidate();
		},
		// 打开新增 PSSC 弹窗
		addPSSCDialogOpen(row,optType,tbType){
			var vm = this;
			vm.tbType = tbType;
			vm.optType = optType;
			if(vm.optType == 'edit'){
				Object.assign(vm.addPSSCDialogForm,row)
			}
			vm.addPSSCDialogShow = true;
		},
		// 新增 PSSC 提交
		addPSSCDialogSubmit(){
			var vm = this,
				params = {},
				codes = {
					'PSSC':'PSSCList',
				},
				idxStr = vm.tbType + '_idx',
				optTb = codes[vm.tbType];
			Object.keys(vm.addPSSCDialogForm).forEach(function(key){
				params[key] = vm.addPSSCDialogForm[key]
			})
			if(vm.addPSSCDialogForm.operateType){
				params.operateType = vm.addPSSCDialogForm.operateType
			}
			vm.$refs.addPSSCDialogForm.validate(function(valid){
				if(valid){
					if(vm.optType == 'add'){
						params.operateType = 'add'
						if(vm.ruleForm[optTb].length == 0){
							params[idxStr] = '1'
						}else{
							var idList=[];
							vm.ruleForm[optTb].map((item)=>{
								idList.push(item[idxStr]);
							})
							params[idxStr] = vm.createId(1,idList); 
						}
						vm.ruleForm[optTb].push(params);
					}else{
						if(params.operateType && params.operateType == 'add'){
							params.operateType = 'add'
						}else{
							params.operateType = 'edit';
						}
						var idx='';
						vm.ruleForm[optTb].map((item,index)=>{
							if(item[idxStr] == params[idxStr]){
								idx = index
							}
						})
						Object.assign(vm.ruleForm[optTb][idx],params)
					}
					vm.addPSSCDialogShow = false;
				}
			})
		},
		// 删除 PSSC
		delPSSCList(row,tbType){
			var vm = this,
				codes = {
					'PSSC':'PSSCList',
				},
				idxStr = tbType + '_idx',
				optTb = codes[tbType];
			var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
			vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(()=>{
				var delFlag=false;
				vm.ruleForm[optTb].map(function(item,index){
					if(item[idxStr] == row[idxStr]){
						if(item.operateType == 'add'){
							delFlag = true;
						}else{
							var params = item;
							params.operateType = 'remove';
							vm.$set(vm.ruleForm[optTb],index,params);
						}
					}
				})
				if(delFlag){
					vm.ruleForm[optTb] = vm.ruleForm[optTb].filter((items)=>{
						return items[idxStr] != row[idxStr]
					})
				}
			})
		},
		// 关闭 PSSC 弹窗
		closeAddPSSCDialog(){
			var vm = this,
				params = {
					PSSC_SearchSpaceID:'',
					PSSC_CoresetID:'',
					PSSC_PdcchSlotPeriodicity:'1',
					PSSC_PdcchSlotOffset:'',
					PSSC_SearchSpaceDuration:'',
					PSSC_PdcchSymbolsInSlot:'',
					PSSC_PdcchCandidatesAggLevel1:'',
					PSSC_PdcchCandidatesAggLevel2:'',
					PSSC_PdcchCandidatesAggLevel4:'',
					PSSC_PdcchCandidatesAggLevel8:'',
					PSSC_PdcchCandidatesAggLevel16:'',
					PSSC_SearchSpaceType:'0',
					PSSC_DciFormat00And10En:'1',
				};
			Object.assign(vm.addPSSCDialogForm,params);
			vm.$refs.addPSSCDialogForm.clearValidate();
		},
		// 打开新增 PDSCHSSC 弹窗
		addPDSCHSSCDialogOpen(row,optType,tbType){
			var vm = this;
			vm.tbType = tbType;
			vm.optType = optType;
			if(vm.optType == 'edit'){
				Object.assign(vm.addPDSCHSSCDialogForm,row)
			}
			vm.addPDSCHSSCDialogShow = true;
		},
		// 新增 PDSCHSSC 提交
		addPDSCHSSCDialogSubmit(){
			var vm = this,
				params = {},
				codes = {
					'PDSCHSSC':'PDSCHSSCList',
				},
				idxStr = vm.tbType + '_idx',
				optTb = codes[vm.tbType];
			Object.keys(vm.addPDSCHSSCDialogForm).forEach(function(key){
				params[key] = vm.addPDSCHSSCDialogForm[key]
			})
			if(vm.addPDSCHSSCDialogForm.operateType){
				params.operateType = vm.addPDSCHSSCDialogForm.operateType
			}
			vm.$refs.addPDSCHSSCDialogForm.validate(function(valid){
				if(valid){
					if(vm.optType == 'add'){
						params.operateType = 'add'
						if(vm.ruleForm[optTb].length == 0){
							params[idxStr] = '1'
						}else{
							var idList=[];
							vm.ruleForm[optTb].map((item)=>{
								idList.push(item[idxStr]);
							})
							params[idxStr] = vm.createId(1,idList); 
						}
						vm.ruleForm[optTb].push(params);
					}else{
						if(params.operateType && params.operateType == 'add'){
							params.operateType = 'add'
						}else{
							params.operateType = 'edit';
						}
						var idx='';
						vm.ruleForm[optTb].map((item,index)=>{
							if(item[idxStr] == params[idxStr]){
								idx = index
							}
						})
						Object.assign(vm.ruleForm[optTb][idx],params)
					}
					vm.addPDSCHSSCDialogShow = false;
				}
			})
		},
		// 删除 PDSCHSSC
		delPDSCHSSCList(row,tbType){
			var vm = this,
				codes = {
					'PDSCHSSC':'PDSCHSSCList',
				},
				idxStr = tbType + '_idx',
				optTb = codes[tbType];
			var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
			vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(()=>{
				var delFlag=false;
				vm.ruleForm[optTb].map(function(item,index){
					if(item[idxStr] == row[idxStr]){
						if(item.operateType == 'add'){
							delFlag = true;
						}else{
							var params = item;
							params.operateType = 'remove';
							vm.$set(vm.ruleForm[optTb],index,params);
						}
					}
				})
				if(delFlag){
					vm.ruleForm[optTb] = vm.ruleForm[optTb].filter((items)=>{
						return items[idxStr] != row[idxStr]
					})
				}
			})
		},
		// 关闭 PDSCHSSC 弹窗
		closeAddPDSCHSSCDialog(){
			var vm = this,
				params = {
					PDSCHSSC_StartSymbol:'',
					PDSCHSSC_Length:'',
					PDSCHSSC_StartSymbolAndLength:'',
				};
			Object.assign(vm.addPDSCHSSCDialogForm,params);
			vm.$refs.addPDSCHSSCDialogForm.clearValidate();
		},
		// 打开新增 PDSCHD 弹窗
		addPDSCHDDialogOpen(row,optType,tbType){
			var vm = this;
			vm.tbType = tbType;
			vm.optType = optType;
			if(vm.optType == 'edit'){
				Object.assign(vm.addPDSCHDDialogForm,row)
			}
			vm.addPDSCHDDialogShow = true;
		},
		// 新增 PDSCHD 提交
		addPDSCHDDialogSubmit(){
			var vm = this,
				params = {},
				codes = {
					'PDSCHD':'PDSCHDList',
				},
				idxStr = vm.tbType + '_idx',
				optTb = codes[vm.tbType];
			Object.keys(vm.addPDSCHDDialogForm).forEach(function(key){
				params[key] = vm.addPDSCHDDialogForm[key]
			})
			if(vm.addPDSCHDDialogForm.operateType){
				params.operateType = vm.addPDSCHDDialogForm.operateType
			}
			vm.$refs.addPDSCHDDialogForm.validate(function(valid){
				if(valid){
					if(vm.optType == 'add'){
						params.operateType = 'add'
						if(vm.ruleForm[optTb].length == 0){
							params[idxStr] = '1'
						}else{
							var idList=[];
							vm.ruleForm[optTb].map((item)=>{
								idList.push(item[idxStr]);
							})
							params[idxStr] = vm.createId(1,idList); 
						}
						vm.ruleForm[optTb].push(params);
					}else{
						if(params.operateType && params.operateType == 'add'){
							params.operateType = 'add'
						}else{
							params.operateType = 'edit';
						}
						var idx='';
						vm.ruleForm[optTb].map((item,index)=>{
							if(item[idxStr] == params[idxStr]){
								idx = index
							}
						})
						Object.assign(vm.ruleForm[optTb][idx],params)
					}
					vm.addPDSCHDDialogShow = false;
				}
			})
		},
		// 删除 PDSCHD
		delPDSCHDList(row,tbType){
			var vm = this,
				codes = {
					'PDSCHD':'PDSCHDList',
				},
				idxStr = tbType + '_idx',
				optTb = codes[tbType];
			var confirmStr = '<%=rb.getString("QueRenShanChu")%>'
			vm.$confirm(confirmStr,'<%=rb.getString("QueRen")%>',{
				confirmButtonText:'<%=rb.getString("QueDing")%>',
				cancelButtonText:'<%=rb.getString("QuXiao")%>',
				type:'warning',
				closeOnClickModal:false
			}).then(()=>{
				var delFlag=false;
				vm.ruleForm[optTb].map(function(item,index){
					if(item[idxStr] == row[idxStr]){
						if(item.operateType == 'add'){
							delFlag = true;
						}else{
							var params = item;
							params.operateType = 'remove';
							vm.$set(vm.ruleForm[optTb],index,params);
						}
					}
				})
				if(delFlag){
					vm.ruleForm[optTb] = vm.ruleForm[optTb].filter((items)=>{
						return items[idxStr] != row[idxStr]
					})
				}
			})
		},
		// 关闭 PDSCHD 弹窗
		closeAddPDSCHDDialog(){
			var vm = this,
				params = {
					PDSCHD_StartSymbol:'',
					PDSCHD_Length:'',
					PDSCHD_StartSymbolAndLength:''
				};
			Object.assign(vm.addPDSCHDDialogForm,params);
			vm.$refs.addPDSCHDDialogForm.clearValidate();
		},
		settingsSubmit(){
			var vm = this;
			var params = {},
				isChanged = isFormChanged(vm.$refs.ruleForm),
				isSync = false;

			if(!isChanged){
				showMsg('prompt_msg','<%=rb.getString("CanShuZhiMeiYouBianHua")%>');
				return;
			}

			vm.$refs.ruleForm.fields.map(function(field){
				var key = vm.getNameByProp(field.prop);

				if(Array.isArray(field.fieldValue)){
					var vList = field.fieldValue.map(function(item){return item}),
						oList = (field.reinitialValue||[]).map(function(item){return item}),
						val = JSON.stringify(vList.sort()),
						orVal = JSON.stringify(oList.sort());

					if(val != orVal) {
						var editList=[],subList=[];
						vList.map((items)=>{
							if(items.operateType){
								if(field.prop == 'PDSCHSSCList'){
									var StartSymbol = items['PDSCHSSC_StartSymbol'] ? parseInt(items['PDSCHSSC_StartSymbol']) : '',
										Length = items['PDSCHSSC_Length'] ? parseInt(items['PDSCHSSC_Length']) : '';
									if((Length - 1) <= 7){
										items['PDSCHSSC_StartSymbolAndLength'] = 14*(Length - 1) + StartSymbol + '';
									}else{
										items['PDSCHSSC_StartSymbolAndLength'] = 14*( 14 - Length + 1) + ( 14 - 1 - StartSymbol) + '';
									}
								}else if(field.prop == 'PDSCHDList'){
									var StartSymbol = items['PDSCHD_StartSymbol'] ? parseInt(items['PDSCHD_StartSymbol']) : '',
										Length = items['PDSCHD_Length'] ? parseInt(items['PDSCHD_Length']) : '';
									if((Length - 1) <= 7){
										items['PDSCHD_StartSymbolAndLength'] = 14*(Length - 1) + StartSymbol + '';
									}else{
										items['PDSCHD_StartSymbolAndLength'] = 14*( 14 - Length + 1) + ( 14 - 1 - StartSymbol) + '';
									}
								}
								editList.push(items)
							}
						})
						editList.map((items)=>{
							if(items.operateType == 'add'){
								Object.keys(items).map((key)=>{
									if(key.slice(-3) == 'idx'){
										delete items[key]
									}
								})
							}
						})
						
						editList.map((items)=>{
							var objs={};
							for(var listVal in items){
								var listKey = vm.getNameByProp(listVal);
								if(listVal != 'Xn_Status' && listVal != 'PDSCHSSC_StartSymbol' && listVal != 'PDSCHSSC_Length' && listVal != 'PDSCHD_StartSymbol' && listVal != 'PDSCHD_Length'){
									objs[listKey] = items[listVal]
								}
							}
							objs.cellIndex = '1';
							objs.bwpIndex = vm.rowDataInfo.DLBWP_idx;
							subList.push(objs)
						})
						params[key] = subList;
					};
				}else{
					if(vm.isNull(field.fieldValue) && vm.isNull(field.reinitialValue)){
						
					}else if(field.fieldValue != field.reinitialValue) {
						var editData={
							cellIndex:vm.cellIndex,
							bwpIndex:vm.rowDataInfo.DLBWP_idx,
							value:field.fieldValue
						}
						params[key] = editData;

					};
				}
			});
			
			vm.$refs.ruleForm.validate(function(valid){
				if(valid) {
					var rowCode = vm.smallCellCode,
						url = '${ctx}/cell/quicksettings/saveParamValue.action?smallCellCode='+rowCode;
					$('#gnbSetting_main').addClass('loading');
					axios.post(url,stringify({"params": JSON.stringify(params)})).then(res=>{
						var data = res.data;
						if(data["success"]){
							vm.$message.success({type:'success',message:'<%=rb.getString("ChengGong")%>'});
							eventBus.$emit('close-gnb-settingPage');
						}else{
							vm.$message.error(data["message"])
						}
						$('#gnbSetting_main').removeClass('loading');
					})
				}
			});
		},
		getNameByProp(prop) {
			var vm = this,
				reg = /^\w*\.\d*\.\w*$/
				key = prop;
			
			if(reg.test(prop)) {
				var mReg = /\.(\d*)\./,
					sufReg = /\.(\w*)$/,
					idx = prop.match(mReg)[1],
					sufStr = prop.match(sufReg)[1];

				vm.codeList.map(function(name){
					var index = vm.indexs[name];
					if(vm.casts[name] == sufStr && index == idx) {
						key = name;
					}
				});
			}else {
				vm.codeList.map(function(name){
					if(vm.casts[name] == prop) {
						key = name;
					}
				});
			}

			return key;
		},
		createId(idVal,list){
			var vm = this,
				val = idVal + '';
			if(list.includes(val) == true){
				idVal += 1 ;
				return vm.createId(idVal,list);
			}else{
				return  idVal + '';
			}
		},
		closeSettings(){
			gnbRanPage.$refs.sharingSlide.hide();
		},
		//校验IP
        isValidIP(ip){
            var reg =  /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/     
            return reg.test(ip);     
        },
        //Ipv6校验 
        isIPv6(str){ 
            var reg = /^([\da-fA-F]{1,4}:){6}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^::([\da-fA-F]{1,4}:){0,4}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:):([\da-fA-F]{1,4}:){0,3}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){2}:([\da-fA-F]{1,4}:){0,2}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){3}:([\da-fA-F]{1,4}:){0,1}((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){4}:((25[0-5]|2[0-4]\d|[01]?\d\d?)\.){3}(25[0-5]|2[0-4]\d|[01]?\d\d?)$|^([\da-fA-F]{1,4}:){7}[\da-fA-F]{1,4}$|^:((:[\da-fA-F]{1,4}){1,6}|:)$|^[\da-fA-F]{1,4}:((:[\da-fA-F]{1,4}){1,5}|:)$|^([\da-fA-F]{1,4}:){2}((:[\da-fA-F]{1,4}){1,4}|:)$|^([\da-fA-F]{1,4}:){3}((:[\da-fA-F]{1,4}){1,3}|:)$|^([\da-fA-F]{1,4}:){4}((:[\da-fA-F]{1,4}){1,2}|:)$|^([\da-fA-F]{1,4}:){5}:([\da-fA-F]{1,4})?$|^([\da-fA-F]{1,4}:){6}:$/
            return reg.test(str);
        },
        //校验子网掩码
        isMask(str){
            var exp=/^(254|252|248|240|224|192|128|0)\.0\.0\.0|255\.(254|252|248|240|224|192|128|0)\.0\.0|255\.255\.(254|252|248|240|224|192|128|0)\.0|255\.255\.255\.(254|252|248|240|224|192|128|0)$/; 
            return exp.test(str); 		
        },
        // 验证输入的是否是数字
        isNumeric(str) {
            if(str.length==0){
                return false;
            }
            for(var i=0;i<str.length;i++){
                if(str.charAt(i)<"0" || str.charAt(i)>"9"){
                    return false;
                }
            }
            return true;  
        },
		syncSettingsClick(){
			var vm = this,
				urls='${ctx}/cell/quicksettings/sync.action',
				params = {
					smallCellCode:vm.smallCellCode
				},
				str = Math.random().toString();
				
			axios.post(urls,stringify(params)).then(res=>{
				var data = res.data;
				if(data["success"]){
					gnbTabSettingVue.changeMain('coreNetwork');
				}else{
					vm.$message.error(data["message"])
				}
			})
		},
		closeBwpDetails(){
			gnbRanPage.$refs.sharingSlide.hide();
		},
		PDSCHSSC_StartSymbolAndLengthChange(val){
			var vm = this;
			var StartSymbol = vm.addPDSCHSSCDialogForm['PDSCHSSC_StartSymbol'];
			var Length = vm.addPDSCHSSCDialogForm['PDSCHSSC_Length'];
			StartSymbol = parseInt(StartSymbol);
			Length = parseInt(Length);
			if(vm.isNumeric(StartSymbol) && vm.isNumeric(Length)){
				if((Length - 1) <= 7){
					vm.addPDSCHSSCDialogForm['PDSCHSSC_StartSymbolAndLength'] = (14*(Length - 1) + StartSymbol) + '';
				}else{
					vm.addPDSCHSSCDialogForm['PDSCHSSC_StartSymbolAndLength'] = 14*( 14 - Length + 1) + ( 14 - 1 - StartSymbol) + '';
				}
			}else{
				vm.addPDSCHSSCDialogForm['PDSCHSSC_StartSymbolAndLength'] = '';
			}
		},
		PDSCHD_StartSymbolAndLengthChange(val){
			var vm = this;
			var StartSymbol = vm.addPDSCHDDialogForm['PDSCHD_StartSymbol'];
			var Length = vm.addPDSCHDDialogForm['PDSCHD_Length'];
			StartSymbol = parseInt(StartSymbol);
			Length = parseInt(Length);
			if(vm.isNumeric(StartSymbol) && vm.isNumeric(Length)){
				if((Length - 1) <= 7){
					vm.addPDSCHDDialogForm['PDSCHD_StartSymbolAndLength'] = (14*(Length - 1) + StartSymbol) + '';
				}else{
					vm.addPDSCHDDialogForm['PDSCHD_StartSymbolAndLength'] = 14*( 14 - Length + 1) + ( 14 - 1 - StartSymbol) + '';
				}
			}else{
				vm.addPDSCHDDialogForm['PDSCHD_StartSymbolAndLength'] = '';
			}
		}
	},
	mounted() {
		eventBus.$off("bwpDetails-init").$on("bwpDetails-init",this.init)
	}
});

</script>
